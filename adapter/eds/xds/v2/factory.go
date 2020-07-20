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

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// TypeURLHandleFunc is a function that used to parse ads type url data
type TypeURLHandleFunc func(client *ADSClient, resp *envoy_api_v2.DiscoveryResponse) error

var typeURLHandleFuncs map[string]TypeURLHandleFunc

// RegisterTypeURLHandleFunc 注册 ads 回调
func RegisterTypeURLHandleFunc(url string, f TypeURLHandleFunc) {
	if typeURLHandleFuncs == nil {
		typeURLHandleFuncs = make(map[string]TypeURLHandleFunc, 10)
	}
	typeURLHandleFuncs[url] = f
}

// HandleTypeURL ads 回调执行
func HandleTypeURL(url string, client *ADSClient, resp *envoy_api_v2.DiscoveryResponse) error {
	if f, ok := typeURLHandleFuncs[url]; ok {
		return f(client, resp)
	}
	return fmt.Errorf("HandleTypeURL url:%s not support", url)
}
