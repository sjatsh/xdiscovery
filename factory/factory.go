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
package factory

import (
    "errors"
    "fmt"

    "github.com/etcd-io/etcd/clientv3"

    "github.com/sjatsh/xdiscovery"
    "github.com/sjatsh/xdiscovery/adapter/consul"
    "github.com/sjatsh/xdiscovery/adapter/eds"
    "github.com/sjatsh/xdiscovery/adapter/etcd"
    "github.com/sjatsh/xdiscovery/discovery"
)

// Kernel 使用的注册发现内核
type Kernel int

const (
    KernelConsul Kernel = iota + 1 // KernelConsul 使用 consul 注册发现
    KernelEds                      // KernelEds 使用eds服务注册发现
    KernelEtcd                     // KernelEtcd 使用etcd服务注册发现
)

// Opts 合并后的配置
type Opts struct {
    Addrs                []string
    ConsulOpts           consul.Opts
    EdsOpts              []eds.Option
    EtcdOpts             *clientv3.Config
    Degrade              xdiscovery.DegradeOpts
    InitHistoryEndpoints map[string]xdiscovery.ServiceList
}

// NewDiscovery 创建服务注册发现对象
func NewDiscovery(kernel Kernel, opts ...Opts) (xdiscovery.Discovery, error) {
    var err error
    var adapter xdiscovery.Adapter

    opt := Opts{}
    if len(opts) > 0 {
        opt = opts[0]
    }
    if len(opt.Addrs) == 0 {
        return nil, errors.New("addr error")
    }

    switch kernel {
    case KernelConsul:
        if len(opt.Addrs) > 0 {
            opt.ConsulOpts.Address = opt.Addrs
        }
        adapter, err = consul.NewConsulAdapter(&opt.ConsulOpts)
        if err != nil {
            return nil, err
        }
    case KernelEds:
        adapter, err = eds.NewEdsAdapter(opt.Addrs[0], opt.EdsOpts...)
        if err != nil {
            return nil, err
        }
    case KernelEtcd:
        etcdOpts := &clientv3.Config{}
        if opt.EtcdOpts != nil {
            etcdOpts = opt.EtcdOpts
            if len(opt.Addrs) > 0 {
                etcdOpts.Endpoints = opt.Addrs
            }
        }
        adapter, err = etcd.NewEtcdAdapter(etcdOpts)
        if err != nil {
            return nil, err
        }
    default:
        return nil, fmt.Errorf("unsupported kernel")
    }
    xDiscovery, err := discovery.NewXDiscovery(adapter, &discovery.Opts{
        Degrade:              &opt.Degrade,
        InitHistoryEndpoints: opt.InitHistoryEndpoints,
    })
    return xDiscovery, err
}
