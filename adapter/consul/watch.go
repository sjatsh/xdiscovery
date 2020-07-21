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
package consul

import (
    "encoding/json"
    "fmt"
    "io"
    "strings"
    "sync"
    "sync/atomic"
    "time"

    "github.com/hashicorp/consul/api"
    "github.com/hashicorp/consul/api/watch"

    "github.com/sjatsh/xdiscovery"
)

var watchVersion uint64

type watcher struct {
    watchVersion uint64
    addr         []string
    service      string
    dc           string
    errCh        chan error
    ready        chan struct{}
    closed       int32
    onUpdateList xdiscovery.OnUpdateList
    writer       *io.PipeWriter

    plan        *watch.Plan
    previousIdx uint64
    lastEntrys  []*api.ServiceEntry // 上一次 consul 获取的列表(不一定更新到缓存)

    compareServiceMu sync.Mutex // 防止 compareService 并发执行
    mu               sync.RWMutex
    services         []*xdiscovery.Service
    version          uint64
}

func newWatcher(opts *Opts, service, dc string, onUpdateList xdiscovery.OnUpdateList) (*watcher, error) {
    w := &watcher{
        watchVersion: atomic.AddUint64(&watchVersion, 1),
        addr:         opts.Address,
        service:      service,
        dc:           dc,
        errCh:        make(chan error, 1),
        ready:        make(chan struct{}),
        onUpdateList: onUpdateList,
        writer:       xdiscovery.Log.Writer(),
    }

    passingOnly := true
    stale := true
    params := make(map[string]interface{})
    params["type"] = "service"
    params["service"] = service
    params["passingonly"] = passingOnly
    params["stale"] = stale
    plan, err := watch.Parse(params)
    if err != nil {
        return nil, err
    }

    plan.Handler = w.handler
    plan.LogOutput = w.writer
    plan.Datacenter = dc
    w.plan = plan

    conf := &api.Config{
        Address:    strings.Join(w.addr, ","),
        Datacenter: dc,
    }
    client, err := api.NewClient(conf)
    if err != nil {
        return nil, fmt.Errorf("Failed to connect to agent: %v", err)
    }
    go func() {
        // 第一次 watch, 未处理 tags
        nodes, _, err := client.Health().ServiceMultipleTags(service, nil, passingOnly, &api.QueryOptions{
            AllowStale: stale,
        })
        if err != nil {
            xdiscovery.Log.Errorf("service:%s first watch err:%v", service, err)
        } else {
            w.handler(0, nodes)
        }
        close(w.ready)

        w.errCh <- w.plan.RunWithConfig(strings.Join(w.addr, ","), &api.Config{
            WaitTime: opts.WatchWaitTime,
        })
    }()
    return w, nil
}

func (w *watcher) Stop() error {
    if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
        return fmt.Errorf("Already stoped")
    }
    w.plan.Stop()
    w.writer.Close()
    return <-w.errCh
}

func (w *watcher) stopNoWait() error {
    if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
        return fmt.Errorf("Already stoped")
    }
    w.plan.Stop()
    return w.writer.Close()
}

func (w *watcher) WaitReady() error {
    select {
    case <-time.After(time.Second):
        // w.Stop()
        return fmt.Errorf("watcher ready timeout")
    case <-w.ready:
    }
    return nil
}

func (w *watcher) handler(idx uint64, data interface{}) {
    entries := data.([]*api.ServiceEntry)
    w.compareServiceMu.Lock()
    if err := w.compareService(entries); err != nil {
        xdiscovery.Log.Errorf("watcher handler compareService error: %v", err)
    }
    w.compareServiceMu.Unlock()
    w.previousIdx = idx
}

func (w *watcher) compareService(current []*api.ServiceEntry) error {
    w.lastEntrys = current
    // 去掉 id 重复的 entry
    currentRepeat := make(map[string]uint64) // 用于去掉重复 id 的 entry (id->max{entry.Service.CreateIndex})
    for _, entry := range current {
        maxIndex := currentRepeat[entry.Service.ID]
        if entry.Service.CreateIndex > maxIndex {
            currentRepeat[entry.Service.ID] = entry.Service.CreateIndex
        }
    }
    var noRepeatCurrent []*api.ServiceEntry
    for i := 0; i < len(current); i++ {
        entry := current[i]
        if entry.Service.CreateIndex == currentRepeat[entry.Service.ID] {
            noRepeatCurrent = append(noRepeatCurrent, entry)
        }
    }
    current = noRepeatCurrent

    services := make([]*xdiscovery.Service, 0, len(current))
    for _, entry := range current {
        services = append(services, buildService(entry))
    }

    w.mu.Lock()
    w.version++
    version := w.version
    w.services = services
    w.mu.Unlock()
    if w.onUpdateList != nil {
        if err := w.onUpdateList(xdiscovery.ServiceList{
            WatchVersion: w.watchVersion,
            Version:      version,
            Services:     services,
        }); err != nil {
            return err
        }
    }
    xdiscovery.Log.Infof("[consul] service:%s update:%d", w.service, len(current))
    return nil
}

func buildService(entry *api.ServiceEntry) *xdiscovery.Service {
    var healthCheck *xdiscovery.HealthCheck
    data := entry.Service.Meta[xdiscovery.HealthCheckMetaKey]
    if len(data) > 0 {
        hc := &xdiscovery.HealthCheck{}
        if err := json.Unmarshal([]byte(data), hc); err != nil {
            xdiscovery.Log.Errorf("entry:%+v meta healthcheck:%s Unmarshal err:%v", entry, data, err)
        }
        healthCheck = hc
    }
    return &xdiscovery.Service{
        ID:          entry.Service.ID,
        Name:        entry.Service.Service,
        Address:     entry.Service.Address,
        Port:        entry.Service.Port,
        Tags:        entry.Service.Tags,
        Weight:      entry.Service.Weights.Passing,
        Meta:        entry.Service.Meta,
        HealthCheck: healthCheck,
    }
}
