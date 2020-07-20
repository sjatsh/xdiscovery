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
    "time"
)

// RegisterOpts 服务注册配置
type RegisterOpts struct {
    CheckTTL      time.Duration // 服务注册后和 agent 心跳间隔(注册中心异常后可检测到), 默认为 Discovery 中的配置
    CheckIP       string        // 健康检查 IP, 默认为注册的服务 IP
    CheckPort     int           // 健康检查 tcp 端口, 默认为注册的服务端口
    CheckInterval time.Duration // 健康检查间隔时间, 默认 5s
    CheckTimeout  time.Duration // 健康检查超时时间, 默认 2s
    CheckHTTP     *HTTPCheck    // http 健康检查,为空则使用 tcp 端口检查
}

// DegradeOpts 降级配置
type DegradeOpts struct {
    Threshold                 float64       // 自我保护阈值, consul 当前获取的列表数/以前的列表 低于这个值进入自我保护, hc 后节点和当前实时节点一致时推出
    PanicThreshold            float64       // 恐慌阈值, 在自我保护状态下, hc 后节点/以前的列表 低于这个值进入，当此比例大于 Threshold 后推出到自我保护状态
    ThresholdContrastInterval time.Duration // 和多久前对比计算,同时也是历史节点删除的窗口(默认 15m)
    EndpointsSaveInterval     time.Duration // 历史节点归档记录间隔(默认 1m)
    Flag                      int32         // 0-开启自我保护; 1-关闭自我保护,使用注册中心的数据, 默认 0
    HealthCheckInterval       time.Duration // 两次全局健康检查最少间隔时间,默认 30s
    PingTimeout               time.Duration // 健康检查超时时间,默认 500ms(如果注册中心返回了健康检查配置，则此配置无效)
}

// WatchOption watch配置
type WatchOption struct {
    DC string
}