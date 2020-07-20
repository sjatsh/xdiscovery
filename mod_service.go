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
package xdiscovery

import (
    "strconv"
    "strings"
)

// ServiceList 服务节点列表
type ServiceList struct {
    WatchVersion uint64  // watcher 版本号(递增)
    Version      uint64  // 服务列表版本号(递增),[0,maxUint32]表示不走降级,[maxUint32+1,maxUint64]表示降级列表,版本号有可能从高到底转换
    RunMode      RunMode // 运行模式，正常或者恢复到了历史版本,变更会刷新缓存(只在 wrap 被使用)
    Services     []*Service
}

// Service 服务信息
type Service struct {
    ID          string            `json:"id"`           // 节点全局唯一标识
    Name        string            `json:"name"`         // 服务名
    Address     string            `json:"address"`      // 节点地址
    Port        int               `json:"port"`         // 节点端口
    Tags        []string          `json:"tags"`         // 节点 tag 属性
    Weight      int               `json:"weight"`       // 节点权重
    Meta        map[string]string `json:"meta"`         // metadata
    HealthCheck *HealthCheck      `json:"health_check"` // 注册到 consul 中的健康检查(返回节点时才会存在,注册时填写无效)
}

func (svc *Service) GetSelfProtectionID() string {
    if len(svc.Name) > 0 && len(svc.Address) > 0 && svc.Port > 0 {
        return svc.Name + "|" + svc.Address + "|" + strconv.FormatInt(int64(svc.Port), 10)
    }
    Log.Errorf("Service data error [%+v]", *svc)
    return svc.ID
}

// Copy 复制 Service 对象
func (svc *Service) Copy() *Service {
    ret := *svc
    if svc.Tags != nil {
        tags := make([]string, len(svc.Tags))
        copy(tags, svc.Tags)
        ret.Tags = tags
    }
    if svc.Meta != nil {
        meta := make(map[string]string)
        for k, v := range svc.Meta {
            meta[k] = v
        }
        ret.Meta = meta
    }
    if svc.HealthCheck != nil {
        newHealthCheck := *svc.HealthCheck
        if len(svc.HealthCheck.Header) > 0 {
            header := make(map[string][]string)
            for k, v := range svc.HealthCheck.Header {
                newl := make([]string, len(v))
                copy(newl, v)
                header[k] = v
            }
            newHealthCheck.Header = header
        }
        ret.HealthCheck = &newHealthCheck
    }
    return &ret
}

// GetTagsAndServiceName 通过域名获取服务名和 tags
func GetTagsAndServiceName(serviceDomain string) (service string, tags []string, err error) {
    if !RegexpService.MatchString(serviceDomain) {
        err = ErrServiceNameInvalid
        return
    }
    tmps := strings.Split(serviceDomain, ".")
    if len(tmps) < 3 {
        err = ErrServiceNameInvalid
        return
    }

    tags = tmps[:len(tmps)-3]
    service = tmps[len(tmps)-3]
    return
}

// CheckServiceOrTagName 检查服务名或者tag是否符合规范
func CheckServiceOrTagName(name string) bool {
    return RegexpServiceName.MatchString(name)
}
