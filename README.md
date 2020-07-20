## 支持特性
1. 双向健康检查(本基础库会主动和consul保活; 且可配置服务健康检查的tcp ip:port, consul 会主动来检测服务健康状态,默认使用服务注册的ip和端口)
2. 本地缓存服务列表,并实时更新服务变化(通过服务版本号O(1)高效判断或注册变化回调通知函数)
3. 支持服务 MetaData 自定义数据携带(服务注册时设置，发现时也会带上)
4. 支持服务权重设置，可以自己实现按权负载均衡
5. consul 集群故障时不影响现有发现的服务列表，且会在集群恢复后主动同步最近状态
6. 支持 consul,eds 等多种注册中心
#
## consul 地址
#### qa环境
http://10.0.1.101:8500
#
## 自我保护模式和注册规范说明

#
## 使用方法
```go
// 初始化
d, err := factory.NewDiscovery(factory.KernelConsul, factory.Opts{
    ConsulOpts: consul.Opts{ // 只有在使用 KernelConsul 时才需要
        Address: "http://127.0.0.1:8500", // consul 地址(必填)
    },
    Degrade: discovery.DegradeOpts{
        Threshold: 0.5, // 阈值, consul 当前获取的列表数/以前的列表 低于这个值进入自我保护
        Flag:      0,   // 0-开启自我保护; 1-关闭自我保护,使用注册中心的数据, 默认 0
    },
    InitHistoryEndpoints: initHistoryEndpoints, // 每个服务对应的初始化数据(进程重启时从其他地方恢复,map 的 key 是服务名)
})

svc := &discovery.Service{
    Name: // ops 应用名(prd/pre环境使用服务名区分具体请遵循服务注册规范)
    Address: // 本机ip
    Port: // 服务端口
    Weight: //100, 节点权重
    Meta: // 跨集群(或者k8s内的流量灰度)流量权重设置,自我保护健康检查等信息，具体需遵守注册规范
}
p, err := d.Register(svc)
// 注销服务
err = p.Deregister()
// 通过负载均衡每次选择下一个节点
lb, err := loadbalance.NewLoadBalance(d) // 只需要全局初始化一次

endpoint, err = lb.Next(service) // 每次请求都调用
```
#
## 服务名规范
### 服务名必须只包含字符[0-9,a-z,A-Z,-],服务名使用ops标识(下划线替换成'-')

#
## 自定义负载均衡器获取服务的两种方式(有自己定制负载均衡需求的才需要使用)
```go
// ### 注意: 自定义负载均衡为了提高性能做了 zero copy ,所有返回的服务列表的指针(以及map,slice等)数据都不要在外部直接修改！否则会导致内部数据错误
// 获取服务列表(会自动 watch 监听变化),可以通过比较版本号来确定服务列表是否被更新了
// 绑定负载均衡器
checkChange := discovery.NewCheckChange(func(list *discovery.ServiceList) (interface{}, error){
    // lb := newLB(list.Services)
    // return lb,nil

    // 这里的代码会在调用 checkChange.GetValue 时节点发生变更时才会回调,用于新建或更新负载均衡对象
})
// 每次请求都调用(效率高,有缓存),并发安全
serviceList, err := d.GetServers(service)
lb, updated, err := checkChange.GetValue(serviceList)
// lb.Next() // 负载均衡器选择下一个节点
/*======*/
```