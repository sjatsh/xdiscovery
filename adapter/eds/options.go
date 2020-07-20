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
	"time"
)

var defaultOptions = Options{
	RefreshDelay:     10 * time.Second,
	AllRefreshDelay:  30 * time.Minute,
	ConnectTimeout:   2 * time.Second,
	WaitReadyTimeout: 1 * time.Second,
}

type Option func(*Options)

type Options struct {
	ServiceCluster   string        // 服务名
	ServiceNode      string        // 服务节点信息
	LocalIP          string        // 本服务 ip
	Env              string        // qa/pre/prd
	Container        string        // vm/k8s
	Datacenter       string        // 所在集群标识
	RefreshDelay     time.Duration // 订阅 Cluster 刷新时间
	AllRefreshDelay  time.Duration // 所有 Cluster 轮询时间
	ConnectTimeout   time.Duration // ads 服务器连接超时时间
	WaitReadyTimeout time.Duration // 等待第一次 ads 配置获取时间
}

// RefreshDelay 定时获取订阅列表时间
func RefreshDelay(refreshDelay time.Duration) Option {
	return func(options *Options) {
		options.RefreshDelay = refreshDelay
	}
}

// AllRefreshDelay 定时获取所有 Cluster 轮询时间
func AllRefreshDelay(allRefreshDelay time.Duration) Option {
	return func(options *Options) {
		options.AllRefreshDelay = allRefreshDelay
	}
}

// ConnectTimeout ads 服务器连接超时时间
func ConnectTimeout(connectTimeout time.Duration) Option {
	return func(options *Options) {
		options.ConnectTimeout = connectTimeout
	}
}

// WaitReadyTimeout ads 等待第一次 ads 配置获取时间
func WaitReadyTimeout(waitReadyTimeout time.Duration) Option {
	return func(options *Options) {
		options.WaitReadyTimeout = waitReadyTimeout
	}
}

// LocalIP 服务对外 ip
func LocalIP(ip string) Option {
	return func(options *Options) {
		options.LocalIP = ip
	}
}

// ServiceCluster 服务名
func ServiceCluster(cluster string) Option {
	return func(options *Options) {
		options.ServiceCluster = cluster
	}
}

// ServiceNode 服务名
func ServiceNode(node string) Option {
	return func(options *Options) {
		options.ServiceNode = node
	}
}

// Env 所在环境
func Env(env string) Option {
	return func(options *Options) {
		options.Env = env
	}
}

// Container 所在 container
func Container(container string) Option {
	return func(options *Options) {
		options.Container = container
	}
}

// Datacenter 所在集群标识
func Datacenter(datacenter string) Option {
	return func(options *Options) {
		options.Datacenter = datacenter
	}
}
