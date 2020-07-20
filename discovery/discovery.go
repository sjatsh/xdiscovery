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
package discovery

import (
    "fmt"
    "sync"
    "sync/atomic"

    "github.com/sjatsh/xdiscovery"
)

var _ xdiscovery.Discovery = (*discoverer)(nil)

// Opts 配置
type Opts struct {
    Degrade              xdiscovery.DegradeOpts
    InitHistoryEndpoints map[string]xdiscovery.ServiceList // 初始化每个服务的历史数据(进程重启时恢复历史节点，否则会在重启时没有历史节点作为基准数据对比)
}

// discoverer xdiscovery
type discoverer struct {
    xdiscovery.Adapter
    initHistoryEndpoints map[string]xdiscovery.ServiceList

    degradeOpts atomic.Value // *xdiscovery.DegradeOpts
    mux         sync.RWMutex
    watchers    map[string]*watch
}

// NewXDiscovery
func NewXDiscovery(opts Opts, adapter xdiscovery.Adapter) (xdiscovery.Discovery, error) {
    d := &discoverer{
        Adapter:              adapter,
        watchers:             map[string]*watch{},
        initHistoryEndpoints: opts.InitHistoryEndpoints,
    }
    if err := d.UpdateDegradeOpts(opts.Degrade); err != nil {
        return nil, err
    }
    return d, nil
}

// GetServers 按照这组服务节点列表和版本号, 请不要修改任何 *ServiceList 中的变量
func (d *discoverer) GetServers(service string, tags ...string) (*xdiscovery.ServiceList, error) {
    return d.GetServersWithDC(service, "", tags...)
}

func (d *discoverer) GetServersWithDC(service string, dc string, tags ...string) (*xdiscovery.ServiceList, error) {
    var key string
    if len(dc) > 0 {
        key = service + "|" + dc
    } else {
        key = service
    }
    d.mux.RLock()
    w := d.watchers[key]
    d.mux.RUnlock()
    newWatch := false
    if w == nil {
        var err error
        d.mux.Lock()
        w = d.watchers[key]
        if w == nil {
            newWatch = true
            w, err = d.newWatch(service, nil, xdiscovery.WatchOption{DC: dc})
            if err != nil {
                d.mux.Unlock()
                xdiscovery.Log.Errorf("[xdiscovery]consul.GetServers err:%v", err)
                return nil, err
            }
            d.watchers[key] = w
        }
        d.mux.Unlock()
    }
    if newWatch {
        xdiscovery.Log.Infof("watch service:%s", service)
    }
    list := w.getList(tags...)
    return list, nil
}

// Watch 监听一个服务变更, 不区分 tag, 直接监听所有 tags
func (d *discoverer) Watch(service string, onUpdate xdiscovery.OnUpdate, opts ...xdiscovery.WatchOption) (xdiscovery.Watcher, error) {
    wop := newWatchOnUpdate(onUpdate)
    return d.WatchList(service, wop.onUpdateList, opts...)
}

// UpdateDegradeOpts 动态更新降级配置
func (d *discoverer) UpdateDegradeOpts(opts xdiscovery.DegradeOpts) error {
    if opts.Flag < 0 || opts.Flag > 1 {
        return fmt.Errorf("invalid flag:%d", opts.Flag)
    }
    if opts.ThresholdContrastInterval == 0 {
        opts.ThresholdContrastInterval = defaultThresholdContrastInterval
    }
    if opts.EndpointsSaveInterval == 0 {
        opts.EndpointsSaveInterval = defaultEndpointsSaveInterval
    }
    if opts.HealthCheckInterval == 0 {
        opts.HealthCheckInterval = defaultHealthCheckInterval
    }
    if opts.PingTimeout == 0 {
        opts.PingTimeout = defaultPingTimeout
    }
    if opts.PanicThreshold == 0 {
        opts.PanicThreshold = defaultPanicThreshold
    }
    d.degradeOpts.Store(&opts)
    var watchers []xdiscovery.Watcher
    d.mux.RLock()
    for _, w := range d.watchers {
        watchers = append(watchers, w)
    }
    d.mux.RUnlock()
    for _, w := range watchers {
        deg := w.(*watch)
        if err := deg.updateOpts(opts); err != nil {
            return err
        }
    }
    return nil
}

// Shutdown 停止
func (d *discoverer) Shutdown() error {
    var watchers []xdiscovery.Watcher
    d.mux.RLock()
    for _, w := range d.watchers {
        watchers = append(watchers, w)
    }
    d.mux.RUnlock()
    for _, w := range watchers {
        if err := w.Stop(); err != nil {
            xdiscovery.Log.Errorf("discoverer watch shutdown err:%v", err)
        }
    }
    return nil
}

// WatchList 包装 Adapter.WatchList, 注入降级策略和 getList
func (d *discoverer) WatchList(service string, onUpdateList xdiscovery.OnUpdateList, opts ...xdiscovery.WatchOption) (xdiscovery.Watcher, error) {
    return d.newWatch(service, onUpdateList, opts...)
}

func (d *discoverer) newWatch(service string, onUpdateList xdiscovery.OnUpdateList, wopts ...xdiscovery.WatchOption) (*watch, error) {
    degradeOpts := d.degradeOpts.Load().(*xdiscovery.DegradeOpts)
    deg, err := newDegrade(service, *degradeOpts, d.initHistoryEndpoints, onUpdateList)
    if err != nil {
        return nil, err
    }
    w, err := d.Adapter.WatchList(service, deg.onUpdateList, wopts...)
    if err != nil {
        return nil, err
    }
    wa := newWatcher(service, deg.wrapWatcher(w))
    return wa, nil
}
