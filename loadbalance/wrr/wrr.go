// Copyright 2020 SunJun <i@sjis.me>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package wrr

import (
    "fmt"
    "math/rand"
    "sync"
    "sync/atomic"
    "time"

    "github.com/sjatsh/xdiscovery"
)

var defaultWeight = 1

// RoundRobinWeight implements dynamic weighted round robin load balancer http handler
type RoundRobinWeight struct {
    mutex sync.RWMutex

    servers    []*server
    serversMap map[string]*server
    serverList atomic.Value // *serverList
}

// NewRoundRobinWeight created a new RoundRobinWeight
func NewRoundRobinWeight() *RoundRobinWeight {
    return &RoundRobinWeight{
        servers:    []*server{},
        serversMap: make(map[string]*server),
    }
}

// NextServer 获取下一个节点
func (r *RoundRobinWeight) NextServer() (*xdiscovery.Service, error) {
    s, _ := r.serverList.Load().(*serverList)
    if s == nil {
        return nil, fmt.Errorf("no endpoints in the pool")
    }
    u := s.nextServer()
    if u == nil {
        return nil, fmt.Errorf("no endpoints in the pool")
    }
    return u.Copy(), nil
}

// UpsertServer In case if server is already present in the load balancer, returns error
func (r *RoundRobinWeight) UpsertServer(u *xdiscovery.Service) error {
    if u == nil {
        return fmt.Errorf("server URL can't be nil")
    }
    srv := &server{service: u.Copy()}
    srv.weight = u.Weight
    if srv.weight == 0 {
        srv.weight = defaultWeight
    }

    r.mutex.Lock()
    defer r.mutex.Unlock()

    if s := r.findServer(srv.service); s != nil {
        if s.weight == 0 {
            s.weight = defaultWeight
        }
        r.resetState()
        return nil
    }

    srv.index = len(r.servers)
    r.servers = append(r.servers, srv)
    r.serversMap[srv.service.ID] = srv
    r.resetState()
    return nil
}

// RemoveServer remove a server
func (r *RoundRobinWeight) RemoveServer(u *xdiscovery.Service) error {
    r.mutex.Lock()

    e := r.findServer(u)
    if e == nil {
        r.mutex.Unlock()
        return fmt.Errorf("server not found")
    }
    for i := e.index + 1; i < len(r.servers); i++ {
        r.servers[i].index--
    }
    r.servers = append(r.servers[:e.index], r.servers[e.index+1:]...)
    delete(r.serversMap, u.ID)
    r.resetState()
    r.mutex.Unlock()
    return nil
}

// Servers gets servers URL
func (r *RoundRobinWeight) Servers() []*xdiscovery.Service {
    r.mutex.RLock()
    defer r.mutex.RUnlock()

    out := make([]*xdiscovery.Service, len(r.servers))
    for i, srv := range r.servers {
        out[i] = srv.service
    }
    return out
}

func (r *RoundRobinWeight) findServer(u *xdiscovery.Service) *server {
    return r.serversMap[u.ID]
}

func (r *RoundRobinWeight) resetState() {
    s := buildServerList(r.servers)
    r.serverList.Store(s)
}

// Set additional parameters for the server can be supplied when adding server
type server struct {
    service *xdiscovery.Service
    // Relative weight for the enpoint to other enpoints in the load balancer
    weight int

    index int // 数组下表
}

// serverList 服务列表选择
type serverList struct {
    servers []*server
    r       *rand.Rand
}

func buildServerList(servers []*server) *serverList {
    s := &serverList{
        r: rand.New(newLockedSource()),
    }
    if len(servers) < 0 {
        return s
    }
    s.servers = make([]*server, 0, len(servers))
    for _, v := range servers {
        if v.weight <= 0 {
            continue
        }
        tmp := *v
        s.servers = append(s.servers, &tmp)
    }
    for i := 1; i < len(servers); i++ {
        s.servers[i].weight += s.servers[i-1].weight
    }
    return s
}

func (s *serverList) nextServer() *xdiscovery.Service {
    n := len(s.servers)
    if n == 0 {
        return nil
    }
    val := s.r.Intn(s.servers[n-1].weight)
    li, ri := 0, n
    for li < ri {
        m := (li + ri) >> 1
        if s.servers[m].weight <= val {
            li = m + 1
        } else if s.servers[m].weight > val {
            ri = m
        }
    }
    return s.servers[li].service
}

type lockedSource struct {
    lk  sync.Mutex
    src rand.Source
}

func newLockedSource() rand.Source {
    return &lockedSource{
        src: rand.NewSource(time.Now().UnixNano()),
    }
}

func (r *lockedSource) Int63() (n int64) {
    r.lk.Lock()
    n = r.src.Int63()
    r.lk.Unlock()
    return
}

func (r *lockedSource) Seed(seed int64) {
    r.lk.Lock()
    r.src.Seed(seed)
    r.lk.Unlock()
}
