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
    return r.register()
}

func (r *registry) init(svc *xdiscovery.Service, opts *xdiscovery.RegisterOpts) error {
    newSvc, newOpts, err := adapter.ParseService(svc, opts)
    if err != nil {
        return err
    }
    r.svc.Store(newSvc)
    r.opts.Store(newOpts)

    // 续租并监听续租状态
    if err := r.setLease(600); err != nil {
        return err
    }
    go r.listenLeaseRespChan()

    maxRetry := 2
    for i := 0; i < maxRetry; i++ {
        if err := r.register(); err == nil {
            break
        }
        if i == maxRetry-1 {
            return fmt.Errorf("service:%+v register err:%v", *svc, err)
        }
        xdiscovery.Log.Errorf("service:%+v register err:%v", *svc, err)
        time.Sleep(time.Second)
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
    return nil
}

func (r *registry) register() error {
    opts := r.opts.Load().(*xdiscovery.RegisterOpts)
    svc := r.svc.Load().(*xdiscovery.Service)

    // 注入健康检查信息
    meta := svc.Meta
    if meta == nil {
        meta = map[string]string{}
    }
    if _, existed := meta[xdiscovery.HealthCheckMetaKey]; !existed {
        b, err := json.Marshal(&xdiscovery.HealthCheck{
            Interval: api.ReadableDuration(opts.CheckInterval),
            Timeout:  api.ReadableDuration(opts.CheckTimeout),
            HTTP:     "http://" + opts.CheckIP + ":" + strconv.Itoa(opts.CheckPort) + opts.CheckHTTP.Path,
            Header:   opts.CheckHTTP.Header,
            Method:   opts.CheckHTTP.Method,
            TCP:      opts.CheckIP + ":" + strconv.Itoa(opts.CheckPort),
        })
        if err != nil {
            return err
        }
        meta[xdiscovery.HealthCheckMetaKey] = string(b)
    }
    meta[xdiscovery.RegTimeMetaKey] = strconv.FormatInt(time.Now().Unix(), 10)

    v := map[string]interface{}{
        "name":    svc.Name,
        "address": svc.Address,
        "port":    svc.Port,
        "weight":  svc.Weight,
        "tags":    svc.Tags,
        "meta":    meta,
    }
    data, _ := json.Marshal(v)

    svc.ID = strings.ReplaceAll(svc.ID, adapter.ConsulServiceIDSplitChar, adapter.EtcdServiceIDSplitChar)

    kv := clientv3.NewKV(r.client)
    lease := clientv3.WithLease(r.leaseResp.ID)
    if _, err := kv.Put(context.TODO(), svc.ID, string(data), lease); err != nil {
        return err
    }
    return nil
}

func (r *registry) listenLeaseRespChan() {
    for {
        select {
        case leaseKeepResp := <-r.keepAliveChan:
            if leaseKeepResp == nil {
                xdiscovery.Log.Warnf("close lease\n")
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
