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
    "errors"
    "sync"
)

// ErrNotExistOnRecover 恢复版本中不存在这个服务
var ErrNotExistOnRecover = errors.New("not exis on recover")

// WrapRecover 恢复历史版本包装器
type WrapRecover interface {
    Discovery
    // GetAllWatchedServices 获得所有通过 GetServers() watch 过的服务
    GetAllWatchedServices() (map[string]ServiceList, error)
    // SetRecover 设置返回的服务列表信息，srvs 为nil时会恢复到 RunModeNormal 状态，否则变成 RunModeRecover,不要修改ServiceList.Services(传入的是引用)
    SetRecover(srvs map[string]ServiceList)
}

// WithRecover 可以恢复历史版本的 discovery
type WithRecoverDiscovery struct {
    Discovery

    mux             sync.RWMutex
    watchedServices map[string]struct{}
    srvs            map[string]ServiceList // 历史版本
}

// WrapWithRecoverDiscovery 包装 Discovery
var WrapWithRecoverDiscovery = func(d Discovery) WrapRecover {
    return &WithRecoverDiscovery{
        Discovery:       d,
        watchedServices: make(map[string]struct{}),
    }
}

// GetServers
func (d *WithRecoverDiscovery) GetServers(service string, tags ...string) (*ServiceList, error) {
    return d.GetServersWithDC(service, "", tags...)
}

// GetServersWithDC 不支持 tags
func (d *WithRecoverDiscovery) GetServersWithDC(service string, dc string, tags ...string) (*ServiceList, error) {
    watched := false
    d.mux.RLock()
    _, watched = d.watchedServices[service]
    srvs := d.srvs
    d.mux.RUnlock()
    if !watched {
        d.mux.Lock()
        d.watchedServices[service] = struct{}{}
        d.mux.Unlock()
    }
    if len(srvs) > 0 { // 使用历史版本
        list, ok := srvs[service]
        if !ok {
            return nil, ErrNotExistOnRecover
        }
        list.RunMode = RunModeRecover
        return &list, nil
    }
    return d.Discovery.GetServersWithDC(service, dc)
}

// GetAllWatchedServices 获得所有通过 GetServers() watch 过的服务
func (d *WithRecoverDiscovery) GetAllWatchedServices() (map[string]ServiceList, error) {
    srvs := map[string]ServiceList{}
    d.mux.RLock()
    for s := range d.watchedServices {
        srv, err := d.GetServersWithDC(s, "")
        if err != nil {
            if err != ErrNotExistOnRecover {
                d.mux.RUnlock()
                return nil, err
            }
            continue
        }
        srvs[s] = *srv
    }
    d.mux.RUnlock()
    return srvs, nil
}

// SetRecover 设置返回的服务列表信息
func (d *WithRecoverDiscovery) SetRecover(srvs map[string]ServiceList) {
    if srvs == nil {
        srvs = map[string]ServiceList{}
    }
    d.mux.Lock()
    d.srvs = srvs
    d.mux.Unlock()
}
