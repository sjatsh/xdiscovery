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
    "os"
    "strconv"
    "sync/atomic"
    "time"

    "github.com/hashicorp/consul/api"

    "github.com/sjatsh/xdiscovery"
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

func (p *registry) parseService(svc *xdiscovery.Service, opts *xdiscovery.RegisterOpts) error {
    newOpt := &xdiscovery.RegisterOpts{
        CheckIP:       svc.Address,
        CheckPort:     svc.Port,
        CheckInterval: DefaultCheckInterval,
        CheckTimeout:  DefaultCheckTimeout,
        CheckTTL:      p.d.opts.TTL,
        CheckHTTP:     opts.CheckHTTP,
    }
    if opts.CheckIP != "" {
        newOpt.CheckIP = opts.CheckIP
    }
    if opts.CheckPort != 0 {
        newOpt.CheckPort = opts.CheckPort
    }
    if opts.CheckInterval != 0 {
        newOpt.CheckInterval = opts.CheckInterval
    }
    if opts.CheckTimeout != 0 {
        newOpt.CheckTimeout = opts.CheckTimeout
    }
    if opts.CheckTTL != 0 {
        newOpt.CheckTTL = opts.CheckTTL
    }

    if !xdiscovery.CheckServiceOrTagName(svc.Name) {
        return fmt.Errorf("service name:%s invalid,can only contain characters [0-9,a-z,A-Z,-]", svc.Name)
    }
    for _, tag := range svc.Tags {
        if !xdiscovery.CheckServiceOrTagName(tag) {
            return fmt.Errorf("tag:%s invalid,can only contain characters [0-9,a-z,A-Z,-]", tag)
        }
    }
    if svc.Weight == 0 {
        return fmt.Errorf("service:%s weight cant not be zero", svc.Name)
    }

    if len(svc.ID) == 0 {
        hostname, _ := os.Hostname()
        svc.ID = svc.Name + "~" + svc.Address + "~" + hostname
    }

    p.opts.Store(newOpt)
    p.svc.Store(svc)
    return nil
}

func (p *registry) init(svc *xdiscovery.Service, opts *xdiscovery.RegisterOpts) error {
    if err := p.parseService(svc, opts); err != nil {
        return err
    }
    maxRetry := 2
    for i := 0; i < maxRetry; i++ {
        if err := p.register(false); err != nil {
            if i == maxRetry-1 {
                return fmt.Errorf("service:%+v register err:%v", *svc, err)
            }
            xdiscovery.Log.Errorf("service:%+v register err:%v", *svc, err)
            time.Sleep(time.Second)
        } else {
            break
        }
    }
    return nil
}

func (p *registry) register(update bool) error {
    opts := p.opts.Load().(*xdiscovery.RegisterOpts)
    svc := p.svc.Load().(*xdiscovery.Service)
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
    meta[xdiscovery.HealthCheckMetaKey] = strconv.FormatInt(time.Now().Unix(), 10)
    // 更新 check 信息时先注销之前的 check
    if update {
        checkService.Status = api.HealthPassing // 健康检查默认为 passing
        if err := p.client.Agent().CheckDeregister(checkService.CheckID); err != nil {
            return err
        }
    }
    return p.client.Agent().ServiceRegister(registration)
}

func (p *registry) unregister() error {
    svc := p.svc.Load().(*xdiscovery.Service)
    return p.client.Agent().ServiceDeregister(svc.ID)
}

func (p *registry) Deregister() error {
    if !atomic.CompareAndSwapInt32(&p.deRegistered, 0, 1) {
        return fmt.Errorf("service has been deregister")
    }
    return p.unregister()
}

func (p *registry) Update(s *xdiscovery.Service, opts ...xdiscovery.RegisterOpts) error {
    if atomic.LoadInt32(&p.deRegistered) == 1 {
        return fmt.Errorf("service has been deregister")
    }
    registerOpts := xdiscovery.RegisterOpts{}
    if len(opts) > 0 {
        registerOpts = opts[0]
    }

    psvc := p.svc.Load().(*xdiscovery.Service)
    svc := s.Copy()
    if svc.ID != psvc.ID || svc.Name != psvc.Name {
        return fmt.Errorf("update the service id and name is not allowed")
    }
    if err := p.parseService(svc, &registerOpts); err != nil {
        return err
    }
    return p.register(true)
}
