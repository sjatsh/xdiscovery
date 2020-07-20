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
package v2

import (
    "sync/atomic"
    "time"

    envoyapiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"

    "github.com/sjatsh/xdiscovery"
    "github.com/sjatsh/xdiscovery/internal/utils"
)

// ClientV2 contains config which v2 module needed
type ClientV2 struct {
    ServiceCluster string
    ServiceNode    string
    ClientIP       string
    Env            string // qa/pre/prd
    Container      string // vm/k8s
    Datacenter     string // 所在集群标识
}

// ADSClient communicated with pilot
type ADSClient struct {
    adsConfig         *ADSConfig
    v2Client          *ClientV2
    close             chan struct{}
    onEndpointsUpdate func([]*envoyapiv2.ClusterLoadAssignment) error

    stoped int32

    subscribedClusters atomic.Value // []string // 订阅的所有 Cluster
}

// NewADSClient new a ADSClient
func NewADSClient(adsConfig ADSConfig, onEndpointsUpdate func([]*envoyapiv2.ClusterLoadAssignment) error) *ADSClient {
    if len(adsConfig.LocalIP) == 0 {
        networkCardName := defaultNetworkCardName
        if adsConfig.DefaultNetWorkCard != "" {
            networkCardName = adsConfig.DefaultNetWorkCard
        }
        var err error
        adsConfig.LocalIP, err = utils.GetLocalIP(networkCardName, utils.IpV4)
        if err != nil {
            xdiscovery.Log.Errorf("NewADSClient err:%v", err)
        }
    }
    c := &ADSClient{
        adsConfig: &adsConfig,
        v2Client: &ClientV2{
            ServiceCluster: adsConfig.ServiceCluster,
            ServiceNode:    adsConfig.ServiceNode,
            ClientIP:       adsConfig.LocalIP,
            Env:            adsConfig.Env,
            Container:      adsConfig.Container,
            Datacenter:     adsConfig.Datacenter,
        },
        close:             make(chan struct{}, 1),
        onEndpointsUpdate: onEndpointsUpdate,
    }
    c.subscribedClusters.Store(make([]string, 0))
    return c
}

// UpdateSubscribedClusters 更新订阅 Clusters
func (adsClient *ADSClient) UpdateSubscribedClusters(subscribedClusters []string) {
    clusters := make([]string, len(subscribedClusters))
    copy(clusters, subscribedClusters)
    adsClient.subscribedClusters.Store(clusters)
}

// Start 启动 ads 客户端, 开始发送和获取
func (adsClient *ADSClient) Start() error {
    var err error
    _, err = adsClient.adsConfig.GetStreamClient()
    if err != nil {
        xdiscovery.Log.Errorf("ads start err:%v", err)
        go func() {
            for {
                if adsClient.isStoped() {
                    return
                }
                if _, err = adsClient.adsConfig.GetStreamClient(); err == nil {
                    adsClient.started()
                    xdiscovery.Log.Infof("ads retry started")
                    return
                }
                xdiscovery.Log.Errorf("ads restart err:%v", err)
                time.Sleep(time.Second)
            }
        }()
        return nil
    }
    adsClient.started()
    return nil
}

func (adsClient *ADSClient) started() {
    go adsClient.sendThread()
    go adsClient.receiveThread()
}

func (adsClient *ADSClient) sendThread() {
    xdiscovery.Log.Debugf("send thread request cds")
    err := adsClient.v2Client.reqEndpoints(adsClient.adsConfig.GetClient(), []string{allClusterName})
    if err != nil {
        xdiscovery.Log.Errorf("send thread request eds fail:%v!auto retry next period", err)
        adsClient.reconnect()
    }

    tall := time.NewTicker(adsClient.adsConfig.AllRefreshDelay)
    t1 := time.NewTicker(adsClient.adsConfig.RefreshDelay)
    defer func() {
        tall.Stop()
        t1.Stop()
    }()
    for {
        select {
        case <-adsClient.close:
            xdiscovery.Log.Debugf("send thread receive graceful shut down signal")
            adsClient.adsConfig.closeADSStreamClient()
            return
        case <-tall.C:
            err := adsClient.v2Client.reqEndpoints(adsClient.adsConfig.GetClient(), []string{allClusterName})
            if err != nil {
                xdiscovery.Log.Errorf("send thread request all cluster fail!auto retry next period err:%v", err)
                adsClient.reconnect()
            }
        case <-t1.C:
            clusters := adsClient.subscribedClusters.Load().([]string)
            err := adsClient.v2Client.reqEndpoints(adsClient.adsConfig.GetClient(), clusters)
            if err != nil {
                xdiscovery.Log.Errorf("send thread request eds fail!auto retry next period err:%v", err)
                adsClient.reconnect() // 如果没有订阅则起到心跳作用
            }
        }
    }
}

func (adsClient *ADSClient) receiveThread() {
    for {
        select {
        case <-adsClient.close:
            xdiscovery.Log.Debugf("receive thread receive graceful shut down signal")
            return
        default:
            resp, err := adsClient.adsConfig.GetClient().Recv()
            if err != nil {
                if !adsClient.isStoped() {
                    xdiscovery.Log.Errorf("get resp err: %v, retry after 1s", err)
                    time.Sleep(time.Second)
                }
                continue
            }
            if err := resp.Validate(); err != nil {
                xdiscovery.Log.Errorf("get resp:%+v validate err:%v", resp, err)
                continue
            }
            if err := HandleTypeURL(resp.TypeUrl, adsClient, resp); err != nil {
                xdiscovery.Log.Errorf("[receiveThread] HandleTypeURL err:%v", err)
            }
        }
    }
}

func (adsClient *ADSClient) reconnect() {
    for {
        if adsClient.isStoped() {
            return
        }
        if err := adsClient.adsConfig.streamClient.generateClient(); err != nil {
            xdiscovery.Log.Errorf("stream client reconnect failed:%v, retry after 1s", err)
            time.Sleep(time.Second)
            continue
        }
        xdiscovery.Log.Infof("stream client reconnected")
        break
    }
}

func (adsClient *ADSClient) isStoped() bool {
    return atomic.LoadInt32(&adsClient.stoped) == 1
}

// Stop adsClient wait for send/receive goroutine graceful exit
func (adsClient *ADSClient) Stop() {
    if !atomic.CompareAndSwapInt32(&adsClient.stoped, 0, 1) {
        xdiscovery.Log.Errorf("ads client stop a stoped client")
        return
    }

    close(adsClient.close)
}
