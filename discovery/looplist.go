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
    "container/list"
    "time"

    "github.com/sjatsh/xdiscovery"
)

type elem struct {
    unixNano int64 // 存入时是时间戳
    list     xdiscovery.ServiceList
}

type loopList struct {
    list *list.List
}

func newLoopList() *loopList {
    return &loopList{
        list: list.New(),
    }
}

func (l *loopList) push(elnow *elem, opts *xdiscovery.DegradeOpts) {
    // 删除过期数据
    l.delTimeout(elnow.unixNano, opts)
    // 距离上次插入数据间隔大于 EndpointsSaveInterval 才允许追加写入，否则覆盖同一个时间区间的值(但是时间戳不变)
    el := l.list.Front()
    if el != nil {
        elv := el.Value.(*elem)
        if elnow.unixNano-elv.unixNano < int64(opts.EndpointsSaveInterval) {
            // // 相同时间区间的更新选择节点数多的记录
            // if len(elv.list.Services) > len(elnow.list.Services) {
            //	return
            // }
            l.list.Remove(el)
            elnow.unixNano = elv.unixNano
        }
    }
    l.list.PushFront(elnow)
}

func (l *loopList) back(opts *xdiscovery.DegradeOpts) *elem {
    now := time.Now().Unix()
    // 删除过期数据
    l.delTimeout(now, opts)
    if l.list.Len() > 0 {
        el := l.list.Back()
        if el == nil {
            return nil
        }
        return el.Value.(*elem)
    }
    return nil
}

func (l *loopList) reset() {
    l.list = list.New()
}

func (l *loopList) delTimeout(now int64, opts *xdiscovery.DegradeOpts) {
    for {
        el := l.list.Back()
        if el == nil {
            break
        }
        pre := el.Prev()
        if pre == nil {
            break
        }
        // 至少保留一个15m前数据, 后一个插入的数据必须已经超过 15 分钟才能删除当前的
        prev := pre.Value.(*elem)
        if now-prev.unixNano <= int64(opts.ThresholdContrastInterval) {
            break
        }
        l.list.Remove(el)
    }
}
