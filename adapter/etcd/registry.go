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
package etcd

import (
    "context"
    "encoding/json"
    "fmt"
    "strconv"
    "strings"
    "sync/atomic"
    "time"

    "github.com/etcd-io/etcd/clientv3"
    "github.com/hashicorp/consul/api"

    "github.com/sjatsh/xdiscovery"
    "github.com/sjatsh/xdiscovery/adapter"
)

type registry struct {
    d      *etcdAdapter
    client *clientv3.Client
    opts   atomic.Value // *xdiscovery.RegisterOpts  注册配置
    svc    atomic.Value // *xdiscovery.Service 注册服务

    lease         clientv3.Lease               // 租约
    leaseResp     *clientv3.LeaseGrantResponse // 租约响应结果
    cancelFunc    func()
    keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
    key           string

    deRegistered int32
}

func newRegistry(d *etcdAdapter) *registry {
    return &registry{
        d:      d,
        client: d.client,
    }
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

func (r *registry) init(svc *xdiscovery.Service, opts *xdiscovery.RegisterOpts) error {
    newSvc, newOpts, err := adapter.ParseService(svc, opts)
    if err != nil {
        return err
    }
    r.svc.Store(newSvc)
    r.opts.Store(newOpts)

    maxRetry := 2
    for i := 0; i < maxRetry; i++ {
        if err := r.register(false); err == nil {
            break
        }
        if i == maxRetry-1 {
            return fmt.Errorf("service:%+v register err:%v", *svc, err)
        }
        xdiscovery.Log.Errorf("service:%+v register err:%v", *svc, err)
        time.Sleep(time.Second)
    }

    // 续租并监听续租状态 默认租约
    var ttl int64 = defaultLeaseTime
    if opts.CheckTTL != 0 {
        ttl = int64(opts.CheckTTL)
    }
    if err := r.setLease(ttl); err != nil {
        return err
    }
    return nil
}

func (r *registry) setLease(ttl int64) error {
    lease := clientv3.NewLease(r.client)

    leaseResp, err := lease.Grant(context.TODO(), ttl)
    if err != nil {
        return err
    }

    ctx, cancelFunc := context.WithCancel(context.TODO())
    leaseRespChan, err := lease.KeepAlive(ctx, leaseResp.ID)
    if err != nil {
        return err
    }

    r.lease = lease
    r.leaseResp = leaseResp
    r.cancelFunc = cancelFunc
    r.keepAliveChan = leaseRespChan

    // 监听租约续租心跳
    go r.listenLeaseRespChan()
    return nil
}

func (r *registry) register(update bool) error {
    opts := r.opts.Load().(*xdiscovery.RegisterOpts)
    svc := r.svc.Load().(*xdiscovery.Service)

    // 注入健康检查信息
    meta := svc.Meta
    if meta == nil {
        meta = map[string]string{}
    }
    if _, existed := meta[xdiscovery.HealthCheckMetaKey]; !existed {
        healthCheck := &xdiscovery.HealthCheck{
            Interval: api.ReadableDuration(opts.CheckInterval),
            Timeout:  api.ReadableDuration(opts.CheckTimeout),
            TCP:      opts.CheckIP + ":" + strconv.Itoa(opts.CheckPort),
        }
        if opts.CheckHTTP != nil {
            healthCheck.Header = opts.CheckHTTP.Header
            healthCheck.Method = opts.CheckHTTP.Method
            healthCheck.HTTP = "http://" + opts.CheckIP + ":" + strconv.Itoa(opts.CheckPort) + opts.CheckHTTP.Path
        }
        b, err := json.Marshal(healthCheck)
        if err != nil {
            return err
        }
        meta[xdiscovery.HealthCheckMetaKey] = string(b)
    }
    meta[xdiscovery.RegTimeMetaKey] = strconv.FormatInt(time.Now().Unix(), 10)
    svc.Meta = meta

    data, _ := json.Marshal(svc)

    kv := clientv3.NewKV(r.client)
    var leaseId clientv3.LeaseID
    if update {
        leaseId = r.leaseResp.ID
    }
    lease := clientv3.WithLease(leaseId)
    if _, err := kv.Put(context.TODO(), strings.ReplaceAll(svc.ID, adapter.ConsulServiceIDSplitChar, adapter.EtcdServiceIDSplitChar), string(data), lease); err != nil {
        return err
    }
    return nil
}

func (r *registry) listenLeaseRespChan() {
    for {
        select {
        case leaseKeepResp := <-r.keepAliveChan:
            if leaseKeepResp == nil {
                xdiscovery.Log.Warnf("close lease listen\n")
                return
            }
            xdiscovery.Log.Infof("lease keep alive success\n")
        }
    }
}

func (r *registry) unregister() error {
    r.cancelFunc()
    if _, err := r.lease.Revoke(context.TODO(), r.leaseResp.ID); err != nil {
        return err
    }
    return nil
}
