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
    "context"
    "encoding/json"
    "fmt"
    "io"
    "sync"
    "sync/atomic"
    "time"

    "github.com/coreos/etcd/mvcc/mvccpb"
    "github.com/etcd-io/etcd/clientv3"

    "github.com/sjatsh/xdiscovery"
)

var watchVersion uint64

type watcher struct {
    watchVersion uint64
    addr         []string
    service      string
    dc           string
    ready        chan struct{}
    closed       int32
    onUpdateList xdiscovery.OnUpdateList
    writer       *io.PipeWriter

    previousIdx uint64

    compareServiceMu sync.Mutex // 防止 compareService 并发执行
    mu               sync.RWMutex
    services         []*xdiscovery.Service
    lastGetResp      *clientv3.GetResponse
    version          uint64

    client *clientv3.Client

    // 服务监听channel
    watchCh    clientv3.WatchChan
    cancelFunc context.CancelFunc
}

func newWatcher(config clientv3.Config, service, dc string, onUpdateList xdiscovery.OnUpdateList) (*watcher, error) {
    w := &watcher{
        watchVersion: atomic.AddUint64(&watchVersion, 1),
        addr:         config.Endpoints,
        service:      service,
        dc:           dc,
        ready:        make(chan struct{}),
        onUpdateList: onUpdateList,
        writer:       xdiscovery.Log.Writer(),
    }

    cli, err := clientv3.New(config)
    if err != nil {
        return nil, err
    }
    w.client = cli

    ctx, cancelFunc := context.WithCancel(context.TODO())
    watchChan := cli.Watch(ctx, service, clientv3.WithPrefix())
    w.cancelFunc = cancelFunc
    w.watchCh = watchChan

    go func() {
        // 第一次
        kv := clientv3.NewKV(cli)
        res, err := kv.Get(context.TODO(), service, clientv3.WithPrefix())
        if err != nil {
            xdiscovery.Log.Errorf("service:%s first watch err:%v", service, err)
        } else {
            if res.Kvs != nil && len(res.Kvs) > 0 {
                if err := w.handler(res); err != nil {
                    xdiscovery.Log.Errorf("service:%s first watch handler, getRespKvs: %v err: %v", res.Kvs, service, err)
                }
            }
        }
        close(w.ready)
    }()
    return w, nil
}

func (w *watcher) Stop() error {
    if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
        return fmt.Errorf("already stoped")
    }
    w.cancelFunc()
    if err := w.client.Close(); err != nil {
        return err
    }
    return w.writer.Close()
}

// 等待首次获取是所有注册服务
func (w *watcher) WaitReady() error {
    select {
    case <-time.After(time.Second):
        return fmt.Errorf("watcher ready timeout")
    case <-w.ready:
    }
    return nil
}

func (w *watcher) handler(getRes *clientv3.GetResponse) error {
    w.compareServiceMu.Lock()
    defer w.compareServiceMu.Unlock()

    w.lastGetResp = getRes
    return w.compareService(getRes.Kvs)
}

func (w *watcher) compareService(current []*mvccpb.KeyValue) error {
    services := make([]*xdiscovery.Service, 0, len(current))
    for _, entry := range current {
        svc, err := buildService(entry)
        if err != nil {
            return err
        }
        services = append(services, svc)
    }

    w.mu.Lock()
    w.version++
    version := w.version
    w.services = services
    w.mu.Unlock()
    if w.onUpdateList != nil {
        if err := w.onUpdateList(xdiscovery.ServiceList{
            WatchVersion: w.watchVersion,
            Version:      version,
            Services:     services,
        }); err != nil {
            return err
        }
    }
    xdiscovery.Log.Infof("[etcd] service:%s update:%d", w.service, len(current))
    return nil
}

func buildService(entry *mvccpb.KeyValue) (*xdiscovery.Service, error) {
    svc := &xdiscovery.Service{}
    if err := json.Unmarshal(entry.Value, svc); err != nil {
        return nil, err
    }
    return svc, nil
}
