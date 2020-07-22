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
    "time"

    "github.com/etcd-io/etcd/clientv3"

    "github.com/sjatsh/xdiscovery"
)

const (
    defaultEtcdAddr    = "127.0.0.1:2379"
    defaultDailTimeout = 5 * time.Second
    defaultLeaseTime   = 600
)

type etcdAdapter struct {
    client *clientv3.Client
    config clientv3.Config
}

// etcd服务注册发现适配器
func NewEtcdAdapter(opts ...*clientv3.Config) (xdiscovery.Adapter, error) {
    opt := clientv3.Config{}
    if len(opts) > 0 {
        opt = *opts[0]
    }
    if len(opt.Endpoints) == 0 {
        opt.Endpoints = []string{defaultEtcdAddr}
        opt.DialTimeout = defaultDailTimeout
    }

    cli, err := clientv3.New(opt)
    if err != nil {
        return nil, err
    }
    d := &etcdAdapter{
        client: cli,
        config: opt,
    }
    return d, nil
}

func (d *etcdAdapter) Register(svc *xdiscovery.Service, opts ...xdiscovery.RegisterOpts) (xdiscovery.Registry, error) {
    registerOpts := xdiscovery.RegisterOpts{}
    if len(opts) > 0 {
        registerOpts = opts[0]
    }
    r := newRegistry(d)
    err := r.init(svc.Copy(), &registerOpts)
    if err != nil {
        return nil, err
    }
    return r, err
}

func (d *etcdAdapter) WatchList(service string, onUpdateList xdiscovery.OnUpdateList, opts ...xdiscovery.WatchOption) (xdiscovery.Watcher, error) {
    opt := xdiscovery.WatchOption{}
    if len(opts) > 0 {
        opt = opts[0]
    }
    return newWatcher(d.config, service, opt.DC, onUpdateList)
}
