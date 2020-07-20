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
package loadbalance

import (
    "fmt"
    "sync"

    "github.com/sjatsh/xdiscovery"
    "github.com/sjatsh/xdiscovery/loadbalance/wrr"
)

// LoadBalancer 负载均衡
type LoadBalancer interface {
    Next(service string, opts ...QueryOpts) (*xdiscovery.Service, error)
}

// QueryOpts 负载均衡查询配置
type QueryOpts struct {
    Tags []string
    DC   string
}

type loadBalance struct {
    d xdiscovery.Discovery

    mu     sync.RWMutex
    checks map[string]*xdiscovery.CheckChange
}

// NewLoadBalance 新建负载均衡器(目前只支持按权随机)
func NewLoadBalance(d xdiscovery.Discovery) (LoadBalancer, error) {
    if d == nil {
        return nil, fmt.Errorf("xdiscovery is nil")
    }

    return &loadBalance{
        d:      d,
        checks: make(map[string]*xdiscovery.CheckChange),
    }, nil
}

// Next 返回某个服务的一个节点
func (l *loadBalance) Next(service string, opts ...QueryOpts) (*xdiscovery.Service, error) {
    l.mu.RLock()
    cc := l.checks[service]
    l.mu.RUnlock()
    if cc == nil {
        l.mu.Lock()
        cc = l.checks[service]
        if cc == nil {
            var err error
            cc, err = xdiscovery.NewCheckChange(func(list *xdiscovery.ServiceList) (interface{}, error) {
                lbNew, err := buildLoadBalance(list)
                if err != nil {
                    return nil, err
                }
                return lbNew, nil
            })
            if err != nil {
                return nil, err
            }
            l.checks[service] = cc
        }
        l.mu.Unlock()
    }

    var tags []string
    var dc string
    if len(opts) > 0 {
        tags = opts[0].Tags
        dc = opts[0].DC
    }
    list, err := l.d.GetServersWithDC(service, dc, tags...)
    if err != nil {
        return nil, err
    }
    val, _, err := cc.GetValue(list)
    if err != nil {
        return nil, err
    }
    lb := val.(*wrr.RoundRobinWeight)
    return lb.NextServer()
}

// buildLoadBalance 创建负载均衡器
func buildLoadBalance(list *xdiscovery.ServiceList) (*wrr.RoundRobinWeight, error) {
    rr := wrr.NewRoundRobinWeight()

    for _, srv := range list.Services {
        // Log.Debugf("Creating server at %s with weight %d", u, srv.Weight)
        if err := rr.UpsertServer(srv); err != nil {
            return nil, fmt.Errorf("error adding server %+v to load balancer: %v", srv, err)
        }
    }
    return rr, nil
}
