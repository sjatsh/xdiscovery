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
    envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"

    "github.com/sjatsh/xdiscovery"
)

// 默认
const (
    EnvoyListener              = "type.googleapis.com/envoy.api.v2.Listener"
    EnvoyCluster               = "type.googleapis.com/envoy.api.v2.Cluster"
    EnvoyClusterLoadAssignment = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
    EnvoyRouteConfiguration    = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
)

func init() {
    // RegisterTypeURLHandleFunc(EnvoyListener, HandleEnvoyListener)
    // RegisterTypeURLHandleFunc(EnvoyCluster, HandleEnvoyCluster)
    RegisterTypeURLHandleFunc(EnvoyClusterLoadAssignment, HandleEnvoyClusterLoadAssignment)
    // RegisterTypeURLHandleFunc(EnvoyRouteConfiguration, HandleEnvoyRouteConfiguration)
}

// HandleEnvoyClusterLoadAssignment parse envoy data to mosn endpoint config
func HandleEnvoyClusterLoadAssignment(client *ADSClient, resp *envoy_api_v2.DiscoveryResponse) error {
    xdiscovery.Log.Debugf("get eds resp,handle it ")
    endpoints := client.v2Client.handleEndpointsResp(resp)
    xdiscovery.Log.Debugf("get %d clusters from EDS", len(endpoints))
    // 更新 endpoints
    if err := client.onEndpointsUpdate(endpoints); err != nil {
        return err
    }
    return nil
}
