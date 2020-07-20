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
    "github.com/hashicorp/consul/api"
)

// HTTPCheck http 健康检查
type HTTPCheck struct {
    Path   string              // 路径
    Method string              // 方法
    Header map[string][]string // 请求头
}

// HealthCheck 返回的健康检查
type HealthCheck struct {
    Interval api.ReadableDuration `json:"interval,omitempty"` // 健康检查间隔时间
    Timeout  api.ReadableDuration `json:"timeout,omitempty"`  // 健康检查超时时间
    HTTP     string               `json:"http,omitempty"`     // http 健康检查完整路径 http://...
    Header   map[string][]string  `json:"header,omitempty"`   // http 健康检查带上的 Header
    Method   string               `json:"method,omitempty"`   // http 健康检查的 Method
    TCP      string               `json:"tcp,omitempty"`      // TCP 健康检查的 ip:port
}
