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
    "context"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "net"
    "net/http"
    "sort"
    "strconv"
    "sync"
    "sync/atomic"
    "time"

    "github.com/sjatsh/xdiscovery"
    "github.com/sjatsh/xdiscovery/internal/metrics"
)

// degradeStatus 状态
type degradeStatus int

const (
    statusNormal         degradeStatus = 0
    statusSelfProtection degradeStatus = 1 // 自我保护状态, 进入后使用15分钟前全量列表+当前consul列表后能通过hc的节点
    statusPanic          degradeStatus = 2 // 恐慌状态, 进入后使用15分钟前全量列表+当前consul列表

    defaultThresholdContrastInterval = 15 * time.Minute
    defaultEndpointsSaveInterval     = 1 * time.Minute
    defaultHealthCheckInterval       = 30 * time.Second
    defaultPingTimeout               = 500 * time.Millisecond
    defaultPanicThreshold            = 0.1

    metaKeyStatus     = "status"
    metaStatusOffline = "offline"
)

var (
    startContinueDoPingHook = func() {}
    endContinueDoPingHook   = func() {}
    addEndpoints            = 1 // 计算降级时先增加的节点数
)

// degrade 单个服务的降级
type degrade struct {
    xdiscovery.Watcher
    cluster      string
    opts         atomic.Value // *xdiscovery.DegradeOpts
    doUpdateList xdiscovery.OnUpdateList

    mux                    sync.RWMutex
    pingCtxCancel          context.CancelFunc
    status                 degradeStatus
    latestAdapterList      xdiscovery.ServiceList // Adapter 返回的最新的正常节点列表
    historyEndpoints       *loopList              // elem 历史节点队列(退出自我保护后会重置)
    stableHistoryEndpoints *loopList              // elem 历史节点队列(一直按照时间更新)
    applyList              xdiscovery.ServiceList // 正在应用的节点列表(Normal状态时和latestAdapterList一致, SelfProtection状态时是 latestAdapterList+latestAdapterList经过健康检查后的节点)
}

func newDegrade(cluster string, opts *xdiscovery.DegradeOpts, initHistoryEndpoints map[string]xdiscovery.ServiceList, onUpdateList xdiscovery.OnUpdateList) (*degrade, error) {
    d := &degrade{
        cluster:                cluster,
        doUpdateList:           onUpdateList,
        status:                 statusNormal,
        latestAdapterList:      xdiscovery.ServiceList{},
        historyEndpoints:       newLoopList(),
        stableHistoryEndpoints: newLoopList(),
        applyList:              xdiscovery.ServiceList{},
    }
    if err := d.updateOpts(opts); err != nil {
        return nil, err
    }
    if initHistoryEndpoints != nil {
        list, ok := initHistoryEndpoints[cluster]
        if ok {
            list.Services, _ = d.getNormalServices(list.Services)
            list.Version = 0
            list.WatchVersion = 0
            list.RunMode = xdiscovery.RunModeInit
            d.applyList = list // 初始化也默认使用历史数据(正常注册中心没有问题的话还是会使用注册中心的数据)
            elemInit := &elem{
                unixNano: time.Now().Unix() - int64(opts.ThresholdContrastInterval),
                list:     list,
            }
            d.historyEndpoints.push(elemInit, opts)
            d.stableHistoryEndpoints.push(elemInit, opts)
        }
    }
    return d, nil
}

func (d *degrade) wrapWatcher(watcher xdiscovery.Watcher) *degrade {
    d.Watcher = watcher
    return d
}

func (d *degrade) getList(tags ...string) *xdiscovery.ServiceList {
    d.mux.RLock()
    applyList := d.applyList
    d.mux.RUnlock()
    return &applyList
}

func (d *degrade) getNormalServices(srvs []*xdiscovery.Service) (normals []*xdiscovery.Service, offlines map[string]*xdiscovery.Service) {
    offlines = map[string]*xdiscovery.Service{}
    for i := 0; i < len(srvs); i++ {
        srv := srvs[i]
        offline := false
        if len(srv.Meta) > 0 {
            if srv.Meta[metaKeyStatus] == metaStatusOffline {
                offline = true
            }
        }
        if !offline {
            normals = append(normals, srv)
        } else {
            offlines[srv.ID] = srv
        }
    }
    return normals, offlines
}

// onUpdateList 包装 watcher 的 onUpdateList
func (d *degrade) onUpdateList(xdiscoveryList xdiscovery.ServiceList) error {
    // 有标记下线的节点直接删除
    var offlinesList map[string]*xdiscovery.Service
    newxdiscoveryList := xdiscoveryList
    newxdiscoveryList.Services, offlinesList = d.getNormalServices(newxdiscoveryList.Services)
    xdiscoveryList = newxdiscoveryList

    opts := d.getOpts()
    var el *elem
    d.mux.Lock()
    el = d.historyEndpoints.back(opts) // 服务刚被 watch 时没有历史节点
    d.latestAdapterList = xdiscoveryList
    status := d.status
    d.mux.Unlock()
    // 非正常模式只更新 latestAdapterList
    if status != statusNormal {
        return nil
    }

    // 判断是否进入自我保护模式
    if opts.Flag == 0 && el != nil { // 如果第一次获取时注册中心故障了，会使用初始化传入的节点
        // 历史节点中未被标记下线的节点数
        onlineCount := 0
        for _, srv := range el.list.Services {
            if _, ok := offlinesList[srv.ID]; !ok {
                onlineCount++
            }
        }
        num := len(xdiscoveryList.Services)
        if num > 0 {
            num += addEndpoints
        }
        now := float64(num) / float64(onlineCount)
        if now < opts.Threshold {
            // 进入自我保护模式, 变更状态，并开始健康检查
            d.startSelfProtection()
            b, _ := json.Marshal(xdiscoveryList)
            xdiscovery.Log.Warnf("[adapter consul]cluster:%s degrade can not update list: %s cause:proportion:%f < %f", d.cluster, string(b), now, opts.Threshold)
            return nil
        }
    }

    // 更新 historyEndpoints
    elnow := &elem{
        unixNano: time.Now().Unix(),
        list:     xdiscoveryList,
    }
    d.mux.Lock()
    d.historyEndpoints.push(elnow, opts)
    d.stableHistoryEndpoints.push(elnow, opts)
    d.mux.Unlock()

    // 退出自我保护模式,并更新 applyList
    if err := d.stopSelfProtection(-1); err != nil {
        return err
    }
    // 执行更新通知
    if d.doUpdateList != nil {
        if err := d.doUpdateList(xdiscoveryList); err != nil {
            return err
        }
    }
    return nil
}

func (d *degrade) getOpts() *xdiscovery.DegradeOpts {
    return d.opts.Load().(*xdiscovery.DegradeOpts)
}

func (d *degrade) updateOpts(opts *xdiscovery.DegradeOpts) error {
    d.opts.Store(opts)
    if opts.Flag == 1 {
        // 关闭降级,主动取消正在运行的 ping
        if err := d.stopSelfProtection(-1); err != nil {
            return err
        }
    }
    return nil
}

// startSelfProtection 进入自我保护模式
func (d *degrade) startSelfProtection() {
    d.mux.Lock()
    status := d.status
    if status != statusNormal {
        d.mux.Unlock()
        return
    }
    ctx, cancel := context.WithCancel(context.Background())
    d.status = statusSelfProtection
    d.pingCtxCancel = cancel
    d.mux.Unlock()
    xdiscovery.Log.Warnf("cluster:%s startSelfProtection", d.cluster)
    go d.pingLoop(ctx)
}

// stopSelfProtection 退出自我保护模式
func (d *degrade) stopSelfProtection(want int) error {
    opts := d.getOpts()
    d.mux.Lock()
    if want >= 0 && len(d.latestAdapterList.Services) != want {
        d.mux.Unlock()
        return fmt.Errorf("adapter num not equal want num:%d", want)
    }
    d.applyList = d.latestAdapterList // 使用 Adapter 的数据
    status := d.status
    if status != statusSelfProtection && status != statusPanic {
        d.mux.Unlock()
        return fmt.Errorf("status:%d is not statusSelfProtection or statusPanic", status)
    }
    d.status = statusNormal
    pingCtxCancel := d.pingCtxCancel
    d.pingCtxCancel = nil
    // 清空历史数据，加入当前节点
    el := &elem{
        unixNano: time.Now().Unix(),
        list:     d.latestAdapterList,
    }
    d.historyEndpoints.reset()
    d.historyEndpoints.push(el, opts)
    d.stableHistoryEndpoints.push(el, opts)
    d.mux.Unlock()
    xdiscovery.Log.Warnf("cluster:%s stopSelfProtection", d.cluster)
    // 取消 ping 循环
    if pingCtxCancel != nil {
        pingCtxCancel()
    }
    return nil
}

func (d *degrade) pingLoop(ctx context.Context) {
    opts := d.getOpts()
    d.mux.RLock()
    healthCheckVersion := d.applyList.Version
    historyEndpoints := d.historyEndpoints.back(opts)
    stableHistoryEndpoints := d.stableHistoryEndpoints.back(opts)
    d.mux.RUnlock()
    historyEndpointsStr, _ := json.Marshal(historyEndpoints)
    stableHistoryEndpointsStr, _ := json.Marshal(stableHistoryEndpoints)
    xdiscovery.Log.Warnf("cluster:%s start pingLoop healthCheckVersion:%d, historyEndpoints:%s stableHistoryEndpoints:%s",
        d.cluster, healthCheckVersion, string(historyEndpointsStr), string(stableHistoryEndpointsStr))
    st := time.Now()
    ti := time.NewTimer(opts.HealthCheckInterval)
    defer func() {
        ti.Stop()
        metrics.EventSet(metrics.EventSelfProtectionTime, d.cluster, 0)
        metrics.EventSet(metrics.EventPanicStatus, d.cluster, 0)
    }()
    if !d.continueDoPing(ctx, healthCheckVersion, historyEndpoints, stableHistoryEndpoints) {
        return
    }
    metrics.EventSet(metrics.EventSelfProtectionTime, d.cluster, time.Now().Sub(st).Seconds())
    for {
        select {
        case <-ctx.Done():
            return
        case <-ti.C:
            // 刷新自我保护进入时间
            metrics.EventSet(metrics.EventSelfProtectionTime, d.cluster, time.Now().Sub(st).Seconds())
            healthCheckVersion++ // 每次 ping 都是更新 checkchange 的缓存
            if !d.continueDoPing(ctx, healthCheckVersion, historyEndpoints, stableHistoryEndpoints) {
                xdiscovery.Log.Warnf("cluster:%s stop pingLoop", d.cluster)
                return
            }
            opts = d.getOpts()
            ti.Reset(opts.HealthCheckInterval)
        }
    }
}

func (d *degrade) continueDoPing(ctx context.Context, healthCheckVersion uint64, el, elstable *elem) (doPing bool) {
    startContinueDoPingHook()
    opts := d.getOpts()
    d.mux.RLock()
    latestAdapterList := d.latestAdapterList
    d.mux.RUnlock()
    defer func() {
        if !doPing {
            // 不需要继续 ping, 在 latestAdapterList 没有变化时退出自我保护模式
            if err := d.stopSelfProtection(len(latestAdapterList.Services)); err != nil {
                xdiscovery.Log.Errorf("cluster:%s stopSelfProtection err:%v", d.cluster, err)
                doPing = true
            }
        }
        endContinueDoPingHook()
    }()
    var latestAdapterListServices []*xdiscovery.Service
    // 聚合历史数据和当前最新的 Adapter 节点
    var endpointds []*xdiscovery.Service
    used := map[string]bool{}
    for i := 0; i < len(latestAdapterList.Services); i++ {
        endpoint := latestAdapterList.Services[i]
        latestAdapterListServices = append(latestAdapterListServices, endpoint)
        if !used[endpoint.GetSelfProtectionID()] {
            used[endpoint.GetSelfProtectionID()] = true
            endpointds = append(endpointds, endpoint)
        }
    }
    var historyEndpoints []*xdiscovery.Service
    if el != nil {
        historyEndpoints = el.list.Services
        for i := 0; i < len(historyEndpoints); i++ {
            endpoint := el.list.Services[i]
            if !used[endpoint.GetSelfProtectionID()] {
                used[endpoint.GetSelfProtectionID()] = true
                endpointds = append(endpointds, endpoint)
            }
        }
    }

    // 开始检查,对比检查检查后的数据是否和 latestAdapterList 一致
    now := time.Now()
    tmps, _ := json.Marshal(endpointds)
    xdiscovery.Log.Infof("degrade start ping... endpointds:%s", string(tmps))
    defer func() {
        xdiscovery.Log.Infof("degrade end ping cost:%s", time.Now().Sub(now))
    }()
    var healthList []*xdiscovery.Service
    for _, endpoint := range endpointds {
        if err := checkHealth(endpoint, opts.PingTimeout); err != nil {
            b, _ := json.Marshal(endpoint)
            xdiscovery.Log.Warnf("cluster:%s endpoint:%s check health err:%v", endpoint.Name, string(b), err)
        } else {
            tmp := endpoint
            healthList = append(healthList, tmp)
        }
    }

    // 获取绝对的 15 分钟前列表
    var stableHistoryEndpoints []*xdiscovery.Service
    if elstable != nil {
        stableHistoryEndpoints = elstable.list.Services
    }

    d.mux.RLock()
    status := d.status
    d.mux.RUnlock()
    // 进入或者退出恐慌状态
    switch status {
    case statusSelfProtection:
        // 自我保护下，hc 后的节点数小于恐慌阈值,则进入恐慌保护,使用15分钟前的全量列表
        if len(historyEndpoints) > 0 && float64(len(healthList))/float64(len(historyEndpoints)) < opts.PanicThreshold {
            d.mux.Lock()
            if ctxIsDone(ctx) {
                d.mux.Unlock()
                return true
            }
            if d.status == statusSelfProtection {
                d.status = statusPanic
                d.applyList = xdiscovery.ServiceList{
                    WatchVersion: d.applyList.WatchVersion,
                    Version:      d.applyList.Version, // 恐慌模式下节点不会更新
                    RunMode:      xdiscovery.RunModePanic,
                    Services:     stableHistoryEndpoints, // 使用绝对的 15 分钟前列表
                }
            }
            d.mux.Unlock()
            metrics.EventSet(metrics.EventPanicStatus, d.cluster, 1)
            xdiscovery.Log.Warnf("cluster:%s healthList(%d) historyEndpoints(%d) into panic status PanicThreshold:%f",
                d.cluster, len(healthList), len(historyEndpoints), opts.PanicThreshold)
            return true
        }
    case statusPanic:
        // 恐慌状态下，hc 后的节点数/绝对15m前节点 大于自我保护阈值,则退回到自我保护状态,否则继续保持状态
        if len(stableHistoryEndpoints) > 0 && float64(len(healthList))/float64(len(stableHistoryEndpoints)) < opts.Threshold {
            return true
        } else {
            // 恢复到自我保护状态, 继续走自我保护退出或者更新列表逻辑
            d.mux.Lock()
            if ctxIsDone(ctx) {
                d.mux.Unlock()
                return true
            }
            if d.status == statusPanic {
                d.status = statusSelfProtection
            }
            status = d.status
            d.mux.Unlock()
            metrics.EventSet(metrics.EventPanicStatus, d.cluster, 0)
            xdiscovery.Log.Warnf("cluster:%s healthList(%d) stableHistoryEndpoints(%d) status into selfProtection Threshold:%f",
                d.cluster, len(healthList), len(stableHistoryEndpoints), opts.Threshold)
        }
    }
    // TODO: 退出到自我保护
    // 监控检查后的历史节点+ Adapter 列表和 Adapter 节点一致则退出自我保护到正常模式
    if status == statusSelfProtection && listSame(healthList, latestAdapterListServices) {
        xdiscovery.Log.Warnf("cluster:%s healthcheck list same", d.cluster)
        return false
    }
    // 健康检查发现和Adapter不一致, 使用健康检查后的列表
    d.mux.Lock()
    if ctxIsDone(ctx) {
        d.mux.Unlock()
        return true
    }
    if d.status == statusSelfProtection {
        d.applyList = xdiscovery.ServiceList{
            WatchVersion: d.applyList.WatchVersion,
            Version:      healthCheckVersion,
            RunMode:      xdiscovery.RunModeSelfProtection,
            Services:     healthList,
        }
    }
    d.mux.Unlock()
    return true
}

func listSame(a []*xdiscovery.Service, b []*xdiscovery.Service) bool {
    if len(a) != len(b) {
        return false
    }
    aa := copyList(a)
    bb := copyList(b)
    sort.Slice(aa, func(i, j int) bool {
        return aa[i].ID < aa[j].ID
    })
    sort.Slice(bb, func(i, j int) bool {
        return bb[i].ID < bb[j].ID
    })
    for i := 0; i < len(aa); i++ {
        if aa[i].ID != bb[i].ID {
            return false
        }
    }
    return true
}

func copyList(a []*xdiscovery.Service) []*xdiscovery.Service {
    b := make([]*xdiscovery.Service, 0, len(a))
    for _, v := range a {
        b = append(b, v)
    }
    return b
}

var tcpCheck = func(addr string, timeout time.Duration) error {
    // TODO 使用连接池，避免大量 TIME_WAIT
    conn, err := net.DialTimeout("tcp", addr, timeout)
    if err != nil {
        return err
    }
    return conn.Close()
}

func checkHealth(e *xdiscovery.Service, defaultTimeout time.Duration) error {
    timeout := defaultTimeout
    addr := e.Address + ":" + strconv.FormatInt(int64(e.Port), 10)
    if e.HealthCheck == nil {
        return tcpCheck(addr, timeout)
    }
    if e.HealthCheck.Timeout > 0 {
        timeout = e.HealthCheck.Timeout.Duration()
    }
    if len(e.HealthCheck.HTTP) == 0 {
        if len(e.HealthCheck.TCP) == 0 {
            return tcpCheck(addr, timeout)
        }
        return tcpCheck(e.HealthCheck.TCP, timeout)
    }

    method := e.HealthCheck.Method
    if len(method) == 0 {
        method = http.MethodGet
    }
    req, err := http.NewRequest(method, e.HealthCheck.HTTP, nil)
    if err != nil {
        return err
    }
    req.Header = e.HealthCheck.Header
    if req.Header == nil {
        req.Header = make(http.Header)
    }
    req.Header.Set("User-Agent", "xdiscovery/0.1 static list check health")
    req.Header.Set("X-Autumn-MeshService", e.Name)
    client := http.Client{
        Timeout: timeout,
    }
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
        return err
    }
    defer func() {
        if resp.Body != nil {
            if err := resp.Body.Close(); err != nil {
                xdiscovery.Log.Errorf("check health error: %v", err)
                return
            }
        }
    }()
    if resp.StatusCode < 200 || resp.StatusCode > 399 {
        return fmt.Errorf("http status: %d", resp.StatusCode)
    }
    if meshService := resp.Header.Get("X-Autumn-Mesh-HealthChecked-Service"); len(meshService) > 0 {
        if meshService != e.Name {
            return fmt.Errorf("response mesh service: %s is not : %s", meshService, e.Name)
        }
    }
    return nil
}

func ctxIsDone(ctx context.Context) bool {
    select {
    case <-ctx.Done():
        return true
    default:
        return false
    }
}
