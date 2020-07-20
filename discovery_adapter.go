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
package xdiscovery

import (
    "regexp"
)

var (
    RegexpService     = regexp.MustCompile("^([0-9,a-z,A-Z,-]+\\.)*([0-9,a-z,A-Z,-]+)\\.service\\.discovery$")
    RegexpServiceName = regexp.MustCompile("^([0-9,a-z,A-Z,-]+)$")
)

// OnUpdate 请不要修改任何 Event 中的变量
type OnUpdate func(Event) error

// OnUpdateList 请不要修改任何 Service 中的变量
type OnUpdateList func(ServiceList) error

// Adapter 注册中心适配器接口
type Adapter interface {
    // Register 注册一个服务 并返回服务注册器
    Register(svc *Service, opts ...RegisterOpts) (Registry, error)
    // WatchList 监听一个服务变更一次性返回整个 list, 不区分 tag, 直接监听所有 tags
    WatchList(service string, onUpdateList OnUpdateList, opts ...WatchOption) (Watcher, error)
}

// Registry 服务注册后用于注销或更新
type Registry interface {
    Deregister() error                               // Deregister 注销服务
    Update(svc *Service, opts ...RegisterOpts) error // Update 更新服务注册信息
}
