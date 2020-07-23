package metrics

import (
    "github.com/prometheus/client_golang/prometheus"

    "github.com/sjatsh/xdiscovery/internal/utils"
)

const (
    // EventSelfProtectionTime 进入自我保护时间
    EventSelfProtectionTime = "eventSelfProtectionTime"
    // 进入恐慌状态
    EventPanicStatus = "panicStatus"
)

var (
    events *prometheus.GaugeVec
)

func init() {
    events = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "discovery_events",
    }, []string{"hostname", "cluster", "event"})
    prometheus.MustRegister(events)
}

// EventSet 事件值设置
func EventSet(event, cluster string, value float64) {
    events.WithLabelValues(utils.HostName(), cluster, event).Set(value)
}
