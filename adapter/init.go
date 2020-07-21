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
package adapter

import (
    "fmt"
    "math/rand"
    "os"
    "time"

    "github.com/sjatsh/xdiscovery"
)

const (
    ConsulServiceIDSplitChar = "~"
    EtcdServiceIDSplitChar   = "/"
)

func init() {
    rand.Seed(time.Now().UnixNano())
}

func ParseService(svc *xdiscovery.Service, opts *xdiscovery.RegisterOpts) (*xdiscovery.Service, *xdiscovery.RegisterOpts, error) {
    newOpt := &xdiscovery.RegisterOpts{
        CheckIP:   svc.Address,
        CheckPort: svc.Port,
        CheckHTTP: opts.CheckHTTP,
    }
    if opts.CheckIP != "" {
        newOpt.CheckIP = opts.CheckIP
    }
    if opts.CheckPort != 0 {
        newOpt.CheckPort = opts.CheckPort
    }
    if opts.CheckInterval != 0 {
        newOpt.CheckInterval = opts.CheckInterval
    }
    if opts.CheckTimeout != 0 {
        newOpt.CheckTimeout = opts.CheckTimeout
    }
    if opts.CheckTTL != 0 {
        newOpt.CheckTTL = opts.CheckTTL
    }

    if !xdiscovery.CheckServiceOrTagName(svc.Name) {
        return nil, nil, fmt.Errorf("service name:%s invalid,can only contain characters [0-9,a-z,A-Z,-]", svc.Name)
    }
    for _, tag := range svc.Tags {
        if !xdiscovery.CheckServiceOrTagName(tag) {
            return nil, nil, fmt.Errorf("tag:%s invalid,can only contain characters [0-9,a-z,A-Z,-]", tag)
        }
    }
    if svc.Weight == 0 {
        return nil, nil, fmt.Errorf("service:%s weight cant not be zero", svc.Name)
    }

    if len(svc.ID) == 0 {
        hostname, _ := os.Hostname()
        svc.ID = svc.Name + ConsulServiceIDSplitChar + svc.Address + ConsulServiceIDSplitChar + hostname
    }
    return svc, newOpt, nil
}
