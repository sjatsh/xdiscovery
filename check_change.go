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
    "sync"
)

// CheckChangeOpts 节点变化配置检查
type CheckChangeOpts struct {
    // AsyncUpdate newValue 是否异步执行,异步执行可能会让 GetValue 获取到旧的value值后才触发更新, 而同步执行会阻塞 GetValue,
    //				建议后端服务节点较多时开启 AsyncUpdate, 否则频繁的上下线节点会导致 GetValue 阻塞较多时间,
    //				超低频访问(几分钟才有一次访问的)的服务不建议开启 AsyncUpdate
    AsyncUpdate bool
}

// CheckChange 检查
type CheckChange struct {
    mu           sync.RWMutex
    watchVersion uint64
    version      uint64
    runMode      RunMode
    value        interface{}
    isUpdate     bool

    newValue    func(list *ServiceList) (interface{}, error)
    asyncUpdate bool
}

// NewCheckChange 新建服务变更检查器,用于GetServers() 返回的服务列表检查是否变更;
// newValue() 是在服务变更时新建的值,比如新的负载均衡器,注意可能存在newValue多次但只一个对象会被引用
func NewCheckChange(newValue func(list *ServiceList) (interface{}, error), opts ...CheckChangeOpts) (*CheckChange, error) {
    asyncUpdate := false
    if len(opts) > 0 {
        asyncUpdate = opts[0].AsyncUpdate
    }

    value, err := newValue(&ServiceList{})
    if err != nil {
        return nil, err
    }
    return &CheckChange{
        value:       value, // 先调用一个空列表,防止 lb 没被创建
        newValue:    newValue,
        asyncUpdate: asyncUpdate,
    }, nil
}

// GetValue 根据服务是否变更返回最新的 value,以及是否发生了变更
// 注意: 启用 CheckChangeOpts.AsyncUpdate 时返回updated为true表示触发了value更新，而本次返回的value还是旧值
func (c *CheckChange) GetValue(list *ServiceList) (value interface{}, updated bool, err error) {

    c.mu.RLock()
    // 检查 lb 是否为最新版
    if c.watchVersion != list.WatchVersion || c.version != list.Version || c.runMode != list.RunMode {
        updated = true
    }
    value = c.value
    c.mu.RUnlock()

    if !updated {
        return
    }

    c.mu.Lock()
    defer c.mu.Unlock()

    // 再次检查 lb 是否为最新版
    if c.watchVersion != list.WatchVersion || c.version != list.Version || c.runMode != list.RunMode {
        if c.asyncUpdate {
            if c.isUpdate {
                updated = false
            } else {
                c.isUpdate = true
                log := Log
                go func() {
                    newValue, err := c.newValue(list)
                    if err != nil {
                        c.changeUpdate(false)
                        log.Errorf("qudiscovery checkchange: async new value err:%v", err)
                        return
                    }
                    c.mu.Lock()
                    c.isUpdate = false
                    c.value = newValue
                    c.watchVersion = list.WatchVersion
                    c.version = list.Version
                    c.runMode = list.RunMode
                    c.mu.Unlock()
                }()
            }
        } else {
            newValue, err := c.newValue(list)
            if err != nil {
                return nil, false, err
            }
            c.value = newValue
            c.watchVersion = list.WatchVersion
            c.version = list.Version
            c.runMode = list.RunMode
        }

    }
    value = c.value // 不管是不是自己更新成功的都获取最新的 lb
    return
}

func (c *CheckChange) changeUpdate(isUpdate bool) {
    c.mu.Lock()
    c.isUpdate = isUpdate
    c.mu.Unlock()
}
