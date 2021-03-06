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
    "strconv"
    "sync/atomic"
    "time"

    "github.com/hashicorp/consul/api"

    "github.com/sjatsh/xdiscovery"
    "github.com/sjatsh/xdiscovery/adapter"
)

const (
    DefaultCheckInterval = 5 * time.Second
    DefaultCheckTimeout  = 2 * time.Second
)

type registry struct {
    d      *consulAdapter
    client *api.Client
    opts   atomic.Value // *xdiscovery.RegisterOpts
    svc    atomic.Value // *xdiscovery.Service

    deRegistered int32
}

func newRegistry(d *consulAdapter) *registry {
    p := &registry{
        d:      d,
        client: d.client,
    }
    return p
}

// Deregister 服务注销
func (r *registry) Deregister() error {
    if !atomic.CompareAndSwapInt32(&r.deRegistered, 0, 1) {
        return fmt.Errorf("service has been deregister")
    }
    return r.unregister()
}

// Update 服务信息更新
func (r *registry) Update(s *xdiscovery.Service, opts ...xdiscovery.RegisterOpts) error {
    if atomic.LoadInt32(&r.deRegistered) == 1 {
        return fmt.Errorf("service has been deregister")
    }
    registerOpts := xdiscovery.RegisterOpts{}
    if len(opts) > 0 {
        registerOpts = opts[0]
    }

    psvc := r.svc.Load().(*xdiscovery.Service)
    svc := s.Copy()
    if svc.ID != psvc.ID || svc.Name != psvc.Name {
        return fmt.Errorf("update the service id and name is not allowed")
    }
    // 更新并存储服务和配置信息
    newSvc, newOpts, err := adapter.ParseService(svc, &registerOpts)
    if err != nil {
        return err
    }
    r.svc.Store(newSvc)
    r.opts.Store(newOpts)
    return r.register(true)
}

// init 服务初始化
func (r *registry) init(svc *xdiscovery.Service, opts *xdiscovery.RegisterOpts) error {
    newSvc, newOpts, err := adapter.ParseService(svc, opts)
    if err != nil {
        return err
    }
    // 存储服务对应配置
    r.svc.Store(newSvc)
    r.opts.Store(newOpts)

    // 进行服务注册 进行两次尝试
    maxRetry := 2
    for i := 0; i < maxRetry; i++ {
        if err := r.register(false); err == nil {
            break
        }
        // 注册服务报错
        if i == maxRetry-1 {
            return fmt.Errorf("service:%+v register err:%v", *svc, err)
        }
        xdiscovery.Log.Errorf("service:%+v register err:%v", *svc, err)
        time.Sleep(time.Second)
    }
    return nil
}

// register 服务注册
func (r *registry) register(update bool) error {
    opts := r.opts.Load().(*xdiscovery.RegisterOpts)
    svc := r.svc.Load().(*xdiscovery.Service)
    unregisterAfter := 24 * time.Hour

    checkService := &api.AgentServiceCheck{
        CheckID:                        "service:" + svc.ID,
        Interval:                       opts.CheckInterval.String(),
        DeregisterCriticalServiceAfter: unregisterAfter.String(),
        Timeout:                        opts.CheckTimeout.String(),
    }
    if opts.CheckHTTP != nil {
        checkService.HTTP = "http://" + opts.CheckIP + ":" + strconv.Itoa(opts.CheckPort) + opts.CheckHTTP.Path
        checkService.Header = opts.CheckHTTP.Header
        checkService.Method = opts.CheckHTTP.Method
    } else {
        checkService.TCP = opts.CheckIP + ":" + strconv.Itoa(opts.CheckPort)
    }

    // 注入健康检查信息
    meta := svc.Meta
    if meta == nil {
        meta = map[string]string{}
    }
    if _, existed := meta[xdiscovery.HealthCheckMetaKey]; !existed {
        b, err := json.Marshal(&xdiscovery.HealthCheck{
            Interval: api.ReadableDuration(opts.CheckInterval),
            Timeout:  api.ReadableDuration(opts.CheckTimeout),
            HTTP:     checkService.HTTP,
            Header:   checkService.Header,
            Method:   checkService.Method,
            TCP:      checkService.TCP,
        })
        if err != nil {
            return err
        }
        meta[xdiscovery.HealthCheckMetaKey] = string(b)
    }
    meta[xdiscovery.RegTimeMetaKey] = strconv.FormatInt(time.Now().Unix(), 10)

    registration := &api.AgentServiceRegistration{
        ID:      svc.ID,
        Name:    svc.Name,
        Tags:    svc.Tags,
        Port:    svc.Port,
        Address: svc.Address,
        Weights: &api.AgentWeights{
            Passing: svc.Weight,
            Warning: svc.Weight,
        },
        Meta:   meta,
        Checks: api.AgentServiceChecks{checkService},
    }
    // 更新 check 信息时先注销之前的 check
    if update {
        checkService.Status = api.HealthPassing // 健康检查默认为 passing
        if err := r.client.Agent().CheckDeregister(checkService.CheckID); err != nil {
            return err
        }
    }
    return r.client.Agent().ServiceRegister(registration)
}

func (r *registry) unregister() error {
    svc := r.svc.Load().(*xdiscovery.Service)
    return r.client.Agent().ServiceDeregister(svc.ID)
}
