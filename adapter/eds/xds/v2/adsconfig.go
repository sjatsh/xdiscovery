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
    "fmt"
    "net"
    "sync"
    "time"

    envoyapiv2core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
    ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
)

const (
    allClusterName         = "$all"
    defaultNetworkCardName = "eth0"
)

// StreamClient grpc 客户端
type StreamClient struct {
    conn    *grpc.ClientConn
    aclient ads.AggregatedDiscoveryServiceClient

    mu     sync.RWMutex
    client ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient
    cancel context.CancelFunc
}

func (sc *StreamClient) generateClient() error {
    ctx, cancel := context.WithCancel(context.Background())
    streamClient, err := sc.aclient.StreamAggregatedResources(ctx)
    if err != nil {
        return fmt.Errorf("fail to create stream client: %v", err)
    }

    sc.mu.Lock()
    if sc.cancel != nil {
        sc.cancel()
    }
    sc.cancel = cancel
    sc.client = streamClient
    sc.mu.Unlock()
    return nil
}

func (sc *StreamClient) getClient() ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient {
    sc.mu.RLock()
    client := sc.client
    sc.mu.RUnlock()
    return client
}

func (sc *StreamClient) close() {
    sc.mu.RLock()
    cancel := sc.cancel
    sc.mu.RUnlock()
    cancel()
    sc.conn.Close()
}

// ADSConfig ADS 客户端配置
type ADSConfig struct {
    ServiceCluster     string // 服务名
    ServiceNode        string // 服务节点信息
    APIType            envoyapiv2core.ApiConfigSource_ApiType
    RefreshDelay       time.Duration // 订阅 Cluster 轮询时间
    AllRefreshDelay    time.Duration // 所有 Cluster 轮询时间
    ADSAddr            string        // ads 服务器地址
    ConnectTimeout     time.Duration // ads 服务器连接超时时间
    LocalIP            string        // 本服务 ip
    Env                string        // qa/pre/prd
    Container          string        // vm/k8s
    Datacenter         string        // 所在集群标识
    DefaultNetWorkCard string        // 默认网卡名

    streamClient *StreamClient
}

// GetStreamClient  返回连接到 ads 的 grpc stream client
func (c *ADSConfig) GetStreamClient() (*StreamClient, error) {
    if c.streamClient != nil {
        return c.streamClient, nil
    }

    sc := &StreamClient{}

    dialer := func(addr string, t time.Duration) (net.Conn, error) {
        timeout := c.ConnectTimeout
        if timeout == 0 {
            timeout = t
        }
        return net.DialTimeout("tcp", addr, timeout)
    }
    conn, err := grpc.Dial(c.ADSAddr, grpc.WithInsecure(), grpc.WithDialer(dialer), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*8)))
    if err != nil {
        return nil, fmt.Errorf("did not connect: %v", err)
    }
    sc.conn = conn
    sc.aclient = ads.NewAggregatedDiscoveryServiceClient(conn)
    if err := sc.generateClient(); err != nil {
        if sc.conn != nil {
            sc.conn.Close()
        }
        return nil, err
    }

    c.streamClient = sc
    return sc, nil
}

// GetClient 获取 AggregatedDiscoveryService_StreamAggregatedResourcesClient
func (c *ADSConfig) GetClient() ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient {
    return c.streamClient.getClient()
}

func (c *ADSConfig) closeADSStreamClient() {
    if c.streamClient == nil {
        return
    }
    c.streamClient.close()
}
