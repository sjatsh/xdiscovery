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
    "errors"

    envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
    envoy_api_v2_core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
    ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
    "github.com/gogo/protobuf/proto"
    structpb "github.com/golang/protobuf/ptypes/struct"

    "github.com/sjatsh/xdiscovery"
)

func (c *ClientV2) reqEndpoints(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient, clusterNames []string) error {
    if streamClient == nil {
        return errors.New("stream client is nil")
    }
    err := streamClient.Send(&envoy_api_v2.DiscoveryRequest{
        VersionInfo:   "",
        ResourceNames: clusterNames,
        TypeUrl:       EnvoyClusterLoadAssignment,
        ResponseNonce: "",
        ErrorDetail:   nil,
        Node: &envoy_api_v2_core1.Node{
            Id:      c.ServiceNode,
            Cluster: c.ServiceCluster,
            Metadata: &structpb.Struct{
                Fields: map[string]*structpb.Value{
                    "CLIENT_IP": &structpb.Value{
                        Kind: &structpb.Value_StringValue{
                            StringValue: c.ClientIP,
                        },
                    },
                    "ENV": &structpb.Value{
                        Kind: &structpb.Value_StringValue{
                            StringValue: c.Env,
                        },
                    },
                    "CONTAINER": &structpb.Value{
                        Kind: &structpb.Value_StringValue{
                            StringValue: c.Container,
                        },
                    },
                    "DATACENTER": &structpb.Value{
                        Kind: &structpb.Value_StringValue{
                            StringValue: c.Datacenter,
                        },
                    },
                },
            },
        },
    })
    if err != nil {
        xdiscovery.Log.Errorf("get endpoints fail: %v", err)
        return err
    }
    return nil
}

func (c *ClientV2) handleEndpointsResp(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.ClusterLoadAssignment {
    lbAssignments := make([]*envoy_api_v2.ClusterLoadAssignment, 0)
    for _, res := range resp.Resources {
        lbAssignment := envoy_api_v2.ClusterLoadAssignment{}
        proto.Unmarshal(res.GetValue(), &lbAssignment)
        lbAssignments = append(lbAssignments, &lbAssignment)
    }
    return lbAssignments
}
