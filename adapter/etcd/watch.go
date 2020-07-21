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
    "io"
    "sync"

    "github.com/etcd-io/etcd/clientv3"

    "github.com/sjatsh/xdiscovery"
)

type watcher struct {
    watchVersion uint64
    addr         []string
    service      string
    dc           string
    errCh        chan error
    ready        chan struct{}
    closed       int32
    onUpdateList xdiscovery.OnUpdateList
    writer       *io.PipeWriter

    previousIdx uint64

    compareServiceMu sync.Mutex // 防止 compareService 并发执行
    mu               sync.RWMutex
    services         []*xdiscovery.Service
    version          uint64

    client *clientv3.Client
}

func newWatcher(config clientv3.Config, service, dc string, onUpdateList xdiscovery.OnUpdateList) *watcher {
    return &watcher{}
}

func (w *watcher) Stop() error {
    return nil
}

func (w *watcher) WaitReady() error {
    return nil
}
