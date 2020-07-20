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
    "sync"

    "github.com/sjatsh/xdiscovery"
)

type watch struct {
    *degrade
    service string
    once    sync.Once
}

func newWatcher(service string, deg *degrade) *watch {
    return &watch{
        service: service,
        degrade: deg,
    }
}

func (w *watch) getList(tags ...string) *xdiscovery.ServiceList {
    var err error
    w.once.Do(func() {
        err = w.WaitReady()
    })
    if err != nil {
        xdiscovery.Log.Errorf("service:%s watch wait ready err:%v,will use static list", w.service, err)
    }
    applyList := w.degrade.getList()
    // 在所有自我保护逻辑之后进行 多 tags 筛选
    if len(tags) > 0 {
        tmpApplyList := *applyList
        // 由于自我保护健康检查可能会改变节点列表，只能在这里做过滤
        var retList []*xdiscovery.Service
        for i := 0; i < len(applyList.Services); i++ {
            srv := applyList.Services[i]
            matched := true
            existTags := make(map[string]bool)
            existTags[""] = true
            for _, tag := range srv.Tags {
                existTags[tag] = true
            }
            for _, tag := range tags {
                if !existTags[tag] {
                    matched = false
                    break
                }
            }
            if matched {
                retList = append(retList, srv)
            }
        }
        tmpApplyList.Services = retList
        applyList = &tmpApplyList
    }
    return applyList
}
