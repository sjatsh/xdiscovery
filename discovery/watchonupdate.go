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
package discovery

import (
    "reflect"

    "github.com/sjatsh/xdiscovery"
)

type watchOnUpdate struct {
    onUpdate xdiscovery.OnUpdate

    previous []*xdiscovery.Service
}

func newWatchOnUpdate(onUpdate xdiscovery.OnUpdate) *watchOnUpdate {
    return &watchOnUpdate{
        onUpdate: onUpdate,
    }
}

func (w *watchOnUpdate) onUpdateList(current xdiscovery.ServiceList) error {
    if w.onUpdate == nil {
        return nil
    }
    previousMap := make(map[string]*xdiscovery.Service)
    for _, entry := range w.previous {
        previousMap[entry.ID] = entry
    }
    currentMap := make(map[string]bool)
    for _, entry := range current.Services {
        currentMap[entry.ID] = true
        lastEntry, exist := previousMap[entry.ID]
        if !exist {
            if err := w.onUpdate(xdiscovery.Event{Action: xdiscovery.ActionAdd, Service: entry}); err != nil {
                return err
            }
        } else if !isSameService(lastEntry, entry) {
            if err := w.onUpdate(xdiscovery.Event{Action: xdiscovery.ActionMod, Service: entry}); err != nil {
                return err
            }
        }
    }
    for id, entry := range previousMap {
        if currentMap[id] {
            continue
        }
        if err := w.onUpdate(xdiscovery.Event{Action: xdiscovery.ActionDel, Service: entry}); err != nil {
            return err
        }
    }
    w.previous = current.Services
    return nil
}

func isSameService(l *xdiscovery.Service, r *xdiscovery.Service) bool {
    return reflect.DeepEqual(l, r)
}
