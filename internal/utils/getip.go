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
package utils

import (
	"fmt"
	"net"
)

type IpType int

const (
	IpV4 IpType = iota + 1
	IpV6
)

// GetLocalIP 获取 eth 网卡的 ip
func GetLocalIP(eth string, ipType IpType) (string, error) {
	inters, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, inter := range inters {
		// 检查ip地址判断是否为 eth
		if inter.Name == eth {
			addrs, err := inter.Addrs()
			if err != nil {
				return "", err
			}

			if len(addrs) < 1 {
				return "", fmt.Errorf("can not found ip")
			}

			for _, addr := range addrs {
				ipNet, _ := addr.(*net.IPNet)
				switch ipType {
				case IpV4:
					if ipv4 := ipNet.IP.To4(); ipv4 != nil {
						return ipv4.String(), nil
					}
				case IpV6:
					if ipv6 := ipNet.IP.To16(); ipv6 != nil {
						return ipv6.String(), nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("can not found %s", eth)
}
