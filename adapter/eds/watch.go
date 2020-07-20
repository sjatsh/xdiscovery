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
package eds

import (
    "fmt"
    "sync"
    "sync/atomic"

    "github.com/sjatsh/xdiscovery"
)

var watchVersion uint64

type watcher struct {
    watchVersion uint64
    cluster      string
    closed       int32
    onUpdateList xdiscovery.OnUpdateList

    mu       sync.RWMutex
    clusters []*xdiscovery.Service
    version  uint64
}

func newWatcher(cluster string, onUpdateList xdiscovery.OnUpdateList, clusters []*xdiscovery.Service) *watcher {
    w := &watcher{
        watchVersion: atomic.AddUint64(&watchVersion, 1),
        cluster:      cluster,
        onUpdateList: onUpdateList,
        clusters:     clusters,
    }
    if len(clusters) > 0 {
        w.version++
    }
    return w
}

func (w *watcher) WaitReady() error {
    return nil
}

func (w *watcher) Stop() error {
    if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
        return fmt.Errorf("Already stoped")
    }
    return nil
}

func (w *watcher) updateService(current []*xdiscovery.Service) error {
    w.mu.Lock()
    w.version++
    version := w.version
    w.clusters = current
    w.mu.Unlock()
    if w.onUpdateList != nil {
        if err := w.onUpdateList(xdiscovery.ServiceList{
            WatchVersion: w.watchVersion,
            Version:      version,
            Services:     current,
        }); err != nil {
            return err
        }
    }
    xdiscovery.Log.Infof("[eds] cluster:%s update:%d", w.cluster, len(current))
    return nil
}
