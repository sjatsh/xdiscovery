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
    "fmt"

    "github.com/sjatsh/xdiscovery"
    "github.com/sjatsh/xdiscovery/adapter/consul"
    "github.com/sjatsh/xdiscovery/adapter/eds"
    "github.com/sjatsh/xdiscovery/discovery"
)

// Kernel 使用的注册发现内核
type Kernel int

const (
    KernelConsul Kernel = iota + 1 // KernelConsul 使用 consul 注册发现
    KernelEds                      // KernelEds 使用eds服务注册发现
)

// Opts 合并后的配置
type Opts struct {
    Addr                 string
    ConsulOpts           consul.Opts
    EdsOpts              []eds.Option
    Degrade              xdiscovery.DegradeOpts
    InitHistoryEndpoints map[string]xdiscovery.ServiceList
}

// NewDiscovery 创建服务注册发现对象
func NewDiscovery(kernel Kernel, opts Opts) (xdiscovery.Discovery, error) {
    var err error
    var adapter xdiscovery.Adapter

    switch kernel {
    case KernelConsul:
        if opts.Addr != "" {
            opts.ConsulOpts.Address = opts.Addr
        }
        adapter, err = consul.NewConsulAdapter(&opts.ConsulOpts)
        if err != nil {
            return nil, err
        }
    case KernelEds:
        adapter, err = eds.NewEdsAdapter(opts.Addr, opts.EdsOpts...)
        if err != nil {
            return nil, err
        }
    default:
        return nil, fmt.Errorf("unsupported kernel")
    }
    xDiscovery, err := discovery.NewXDiscovery(discovery.Opts{
        Degrade:              opts.Degrade,
        InitHistoryEndpoints: opts.InitHistoryEndpoints,
    }, adapter)
    return xDiscovery, err
}
