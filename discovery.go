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

// Discovery 服务发现接口
type Discovery interface {
    Adapter

    // Watch 监听一个服务变更, 不区分 tag, 直接监听所有 tags
    Watch(service string, onUpdate OnUpdate, opts ...WatchOption) (Watcher, error)

    // GetServers 按照这组服务节点列表和版本号, 请不要修改任何 *ServiceList 中的变量
    GetServers(service string, tags ...string) (*ServiceList, error)
    GetServersWithDC(service string, dc string, tags ...string) (*ServiceList, error)

    UpdateDegradeOpts(opts DegradeOpts) error // UpdateDegradeOpts  动态更新降级配置, DegradeOpts 具体降级配置
    Shutdown() error                          // Shutdown 停止
}

// Watcher 单个服务事件监听
type Watcher interface {
    Stop() error
    WaitReady() error
}
