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
package etcd

import (
    "fmt"
    "testing"

    "github.com/sjatsh/xdiscovery"
    "github.com/sjatsh/xdiscovery/discovery"
)

func TestEtcd(t *testing.T) {
    adapter, err := NewEtcdAdapter()
    if err != nil {
        t.Fatal(err)
    }
    d, err := discovery.NewXDiscovery(adapter)
    if err != nil {
        t.Fatal(err)
    }
    _, err = d.Watch("adc-def", func(event xdiscovery.Event) error {
        fmt.Printf("11111111111:%v", event)
        return nil
    })
    if err != nil {
        t.Fatal(err)
    }

    _, err = d.Register(&xdiscovery.Service{
        Name:    "abc-def",
        Address: "127.0.0.1",
        Port:    9981,
        Weight:  100,
        HealthCheck: &xdiscovery.HealthCheck{
            HTTP:   "http://127.0.0.1:9981/ping",
            Header: nil,
            Method: "GET",
            TCP:    "127.0.0.1:9981",
        },
    })
    if err != nil {
        t.Fatal(err)
    }

    services, err := d.GetServers("abc-def")
    if err != nil {
        t.Fatal(err)
    }
    fmt.Printf("222222222222222222:%v", services)
}
