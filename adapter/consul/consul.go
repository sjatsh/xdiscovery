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
package consul

import (
    "fmt"
    "math/rand"
    "net"
    "net/url"
    "os"
    "strings"
    "time"

    "github.com/hashicorp/consul/api"
    "github.com/lafikl/consistent"

    "github.com/sjatsh/xdiscovery"
)

var _ xdiscovery.Adapter = (*consulAdapter)(nil)

const (
    defaultAddress               = "http://127.0.0.1:8500"
    defaultTTL                   = time.Minute
    defaultResponseHeaderTimeout = time.Second * 15
    defaultWatchWaitTime         = time.Second * 50
)

type consulAdapter struct {
    opts   Opts
    client *api.Client
    urls   []*url.URL
}

// Opts Opts
type Opts struct {
    Address               string        // consul 或者 agent 地址(多个地址则使用','分割)
    TTL                   time.Duration // 服务注册后和 agent 心跳间隔, 默认5s
    ResponseHeaderTimeout time.Duration // http 请求等待回复时间, 默认15s
    WatchWaitTime         time.Duration // watch 无变每次长轮询等待时间, 默认 50s
}

// NewConsulAdapter NewConsulAdapter
func NewConsulAdapter(opts *Opts) (xdiscovery.Adapter, error) {
    opt := Opts{}
    if opts != nil {
        opt = *opts
    }
    if len(opt.Address) == 0 {
        opt.Address = defaultAddress
    }
    if opt.TTL == 0 {
        opt.TTL = defaultTTL
    }
    if opt.ResponseHeaderTimeout == 0 {
        opt.ResponseHeaderTimeout = defaultResponseHeaderTimeout
    }
    if opt.WatchWaitTime == 0 {
        opt.WatchWaitTime = defaultWatchWaitTime
    }
    addrs := strings.Split(opt.Address, ",")
    var urls []*url.URL
    for _, addr := range addrs {
        u, err := url.Parse(addr)
        if err != nil {
            return nil, err
        }
        urls = append(urls, u)
    }
    d := &consulAdapter{
        opts: opt,
        urls: urls,
    }
    c, err := d.newClient()
    if err != nil {
        return nil, err
    }
    d.client = c
    return d, nil
}

func (d *consulAdapter) Register(svc *xdiscovery.Service, opts ...xdiscovery.RegisterOpts) (xdiscovery.Registry, error) {
    registerOpts := xdiscovery.RegisterOpts{}
    if len(opts) > 0 {
        registerOpts = opts[0]
    }
    r := newRegistry(d)
    err := r.init(svc.Copy(), &registerOpts)
    if err != nil {
        return nil, err
    }
    return r, err
}

func (d *consulAdapter) WatchList(service string, onUpdateList xdiscovery.OnUpdateList, opts ...xdiscovery.WatchOption) (xdiscovery.Watcher, error) {
    opt := xdiscovery.WatchOption{}
    if len(opts) > 0 {
        opt = opts[0]
    }

    return newWatcher(&d.opts, service, opt.DC, onUpdateList)
}

// newClient 随机选择一个 ip 返回新的 consul client
func (d *consulAdapter) newClient() (*api.Client, error) {
    if len(d.urls) == 0 {
        return nil, fmt.Errorf("urls is nil")
    }
    var u url.URL
    if len(d.urls) > 1 {
        u = *d.urls[rand.Intn(len(d.urls))]
    } else {
        host := d.urls[0].Hostname()
        port := d.urls[0].Port()
        scheme := d.urls[0].Scheme + "://"
        ns, err := net.LookupHost(host)
        if err != nil {
            return nil, err
        }
        // hash 选择一个节点
        c := consistent.New()
        for _, u := range ns {
            c.Add(u + ":" + port)
        }
        hostname, _ := os.Hostname()
        host, err = c.Get(hostname)
        if err != nil {
            return nil, fmt.Errorf("hash get err:%v", err)
        }
        ul, err := url.Parse(scheme + host)
        if err != nil {
            return nil, fmt.Errorf("url parse err:%v", err)
        }
        u = *ul
    }
    config := api.DefaultConfig()
    config.Address = u.String()
    config.Transport.ResponseHeaderTimeout = d.opts.ResponseHeaderTimeout
    xdiscovery.Log.Infof("consul client config:%+v", config)
    return api.NewClient(config)
}

func init() {
    rand.Seed(time.Now().UnixNano())
}
