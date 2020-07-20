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

// RunMode 发现库运行模式
type RunMode int32

const (
    RunModeNormal         RunMode = iota + 1 // RunModeNormal 正常的服务发现
    RunModeRecover                           // RunModeRecover 使用本地文件
    RunModeSelfProtection                    // RunModeSelfProtection 节点处于自我保护模式
    RunModeInit                              // RunModeInit 节点处于初始化数据
    RunModePanic                             // RunModePanic 节点处于恐慌状态
)
