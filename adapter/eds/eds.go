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
package eds

import (
    "encoding/json"
    "fmt"
    "reflect"
    "sort"
    "sync"
    "sync/atomic"
    "time"

    envoyapiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
    envoyapiv2core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
    structpb "github.com/golang/protobuf/ptypes/struct"

    "github.com/sjatsh/xdiscovery"
    ads "github.com/sjatsh/xdiscovery/adapter/eds/xds/v2"
)

const (
    edsIDKey = "id"
)

var _ xdiscovery.Adapter = (*Adapter)(nil)

// Adapter EDS 聚合服务发现适配器
type Adapter struct {
    asdClient *ads.ADSClient
    opts      Options
    once      sync.Once
    ready     chan struct{}

    mu          sync.RWMutex
    watchers    map[string]*watcher // cluster->*watcher
    clusterlist []string            // 订阅服务列表
    clusters    atomic.Value        // map[string][]*disocvery.Service 每个 cluster 排序好后的 endpoints
}

// NewAdapter eds 聚合服务发现 Adapter
func NewEdsAdapter(adsAddr string, opts ...Option) (*Adapter, error) {
    options := defaultOptions
    for _, opt := range opts {
        opt(&options)
    }

    d := &Adapter{
        opts:     options,
        ready:    make(chan struct{}),
        watchers: map[string]*watcher{},
    }
    d.clusters.Store(make(map[string][]*xdiscovery.Service))
    c := ads.NewADSClient(ads.ADSConfig{
        APIType:         envoyapiv2core.ApiConfigSource_GRPC,
        ADSAddr:         adsAddr,
        RefreshDelay:    options.RefreshDelay,
        AllRefreshDelay: options.AllRefreshDelay,
        ConnectTimeout:  options.ConnectTimeout,
        ServiceNode:     options.ServiceNode,
        ServiceCluster:  options.ServiceCluster,
        LocalIP:         options.LocalIP,
        Env:             options.Env,
        Container:       options.Container,
        Datacenter:      options.Datacenter,
    }, d.onEndpointsUpdate)
    d.asdClient = c
    return d, nil
}

// Init 初始化确保更新订阅列表
func (d *Adapter) Init(subscribedClusters []string) error {
    d.asdClient.UpdateSubscribedClusters(subscribedClusters)
    if err := d.asdClient.Start(); err != nil {
        return err
    }
    // 等待第一次获取到列表信息
    ti := time.NewTimer(d.opts.WaitReadyTimeout)
    select {
    case <-d.ready:
    case <-ti.C:
        // 等待超时不影响初始化
        xdiscovery.Log.Errorf("eds xdiscovery wait ready timeout")
    }
    return nil
}

// Shutdown 停止监听
func (d *Adapter) Shutdown() error {
    d.asdClient.Stop()
    return nil
}

// Register eds 不支持注册服务
func (d *Adapter) Register(svc *xdiscovery.Service, opts ...xdiscovery.RegisterOpts) (xdiscovery.Registry, error) {
    return nil, fmt.Errorf("unsupported function")
}

// WatchList WatchList
func (d *Adapter) WatchList(cluster string, onUpdateList xdiscovery.OnUpdateList, opts ...xdiscovery.WatchOption) (xdiscovery.Watcher, error) {
    d.mu.RLock()
    w := d.watchers[cluster]
    d.mu.RUnlock()
    if w != nil {
        return nil, fmt.Errorf("cluster:%s already watched", cluster)
    }
    var clusterlist []string
    d.mu.Lock()
    w = d.watchers[cluster]
    if w != nil {
        d.mu.Unlock()
        return nil, fmt.Errorf("cluster:%s already watched", cluster)
    }
    clusters := d.clusters.Load().(map[string][]*xdiscovery.Service) // 使用全部服务列表来初始化 watch
    w = newWatcher(cluster, onUpdateList, clusters[cluster])
    d.watchers[cluster] = w
    d.clusterlist = append(d.clusterlist, cluster)
    clusterlist = d.clusterlist
    d.mu.Unlock()
    // 第一次 watch 时，如果有列表，则更新一次
    if len(clusters) > 0 {
        w.updateService(clusters[cluster])
    }
    if len(clusterlist) > 0 {
        d.asdClient.UpdateSubscribedClusters(clusterlist)
        xdiscovery.Log.Infof("[eds] watch cluster:%s clusterlist:%v", w.cluster, clusterlist)
    }
    return w, nil
}

func (d *Adapter) onEndpointsUpdate(clusters []*envoyapiv2.ClusterLoadAssignment) error {
    defer d.once.Do(func() {
        close(d.ready)
    })

    newClusters := make(map[string][]*xdiscovery.Service)
    changedClusters := make(map[string][]*xdiscovery.Service)
    // 判断每个 Cluster 是否变化
    oldClusters := d.clusters.Load().(map[string][]*xdiscovery.Service)
    for _, c := range clusters {
        var s []*xdiscovery.Service
        for _, v := range c.Endpoints {
            for _, e := range v.LbEndpoints {
                endpoint := e.GetEndpoint()
                if endpoint == nil {
                    return fmt.Errorf("onEndpointsUpdate cluster:%s endpoint:%+v endpoint is nil", c.ClusterName, e)
                }
                addr := endpoint.GetAddress()
                if addr == nil {
                    return fmt.Errorf("onEndpointsUpdate cluster:%s endpoint:%+v addr is nil", c.ClusterName, e)
                }
                sockAddr := addr.GetSocketAddress()
                if sockAddr == nil {
                    return fmt.Errorf("onEndpointsUpdate cluster:%s endpoint:%+v sockAddr is nil", c.ClusterName, e)
                }
                weight := e.GetLoadBalancingWeight()
                if weight == nil {
                    return fmt.Errorf("onEndpointsUpdate cluster:%s endpoint:%+v weight is nil", c.ClusterName, e)
                }
                var (
                    healthcheck *xdiscovery.HealthCheck
                    id          string
                )
                stringMeta := map[string]string{}
                meta := e.GetMetadata()
                if meta != nil && meta.GetFilterMetadata() != nil {
                    metadata := meta.GetFilterMetadata()["istio"]
                    fileds := metadata.GetFields()
                    if metadata != nil && fileds != nil {
                        for k, v := range fileds {
                            if s, ok := v.GetKind().(*structpb.Value_StringValue); ok {
                                stringMeta[k] = s.StringValue
                            }
                        }
                    }
                }
                data := stringMeta[xdiscovery.HealthCheckMetaKey]
                if len(data) > 0 {
                    hc := &xdiscovery.HealthCheck{}
                    if err := json.Unmarshal([]byte(data), hc); err != nil {
                        xdiscovery.Log.Errorf("onEndpointsUpdate cluster:%s endpoint:%+v healthcheck:%s Unmarshal err:%v", c.ClusterName, e, data, err)
                    } else {
                        healthcheck = hc
                    }
                }
                id = stringMeta[edsIDKey]
                if len(id) == 0 {
                    return fmt.Errorf("onEndpointsUpdate cluster:%s endpoint:%+v id is empty", c.ClusterName, e)
                }
                s = append(s, &xdiscovery.Service{
                    ID:          id,
                    Name:        c.ClusterName,
                    Address:     sockAddr.GetAddress(),
                    Port:        int(sockAddr.GetPortValue()),
                    Weight:      int(weight.GetValue()),
                    HealthCheck: healthcheck,
                    Meta:        stringMeta,
                })
            }
        }
        sort.Slice(s, func(i, j int) bool {
            if s[i].Address != s[j].Address {
                return s[i].Address < s[j].Address
            }
            if s[i].Port != s[j].Port {
                return s[i].Port < s[j].Port
            }
            return s[i].Weight < s[j].Weight
        })
        old := oldClusters[c.ClusterName]
        if !reflect.DeepEqual(s, old) {
            changedClusters[c.ClusterName] = s
        }
        newClusters[c.ClusterName] = s
    }
    // 写入新的数据不存在，旧数据存在的服务节点 cluster
    for name, list := range oldClusters {
        if _, ok := newClusters[name]; !ok {
            newClusters[name] = list
        }
    }
    if len(changedClusters) > 0 {
        d.clusters.Store(newClusters)
        for cluster, list := range changedClusters {
            d.mu.RLock()
            w := d.watchers[cluster]
            d.mu.RUnlock()
            if w != nil {
                if err := w.updateService(list); err != nil {
                    return err
                }
            }
        }
    }
    return nil
}
