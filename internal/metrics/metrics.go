package metrics

import (
    "os"

    "github.com/prometheus/client_golang/prometheus"
)

const (
    // EventSelfProtectionTime 进入自我保护时间
    EventSelfProtectionTime = "eventSelfProtectionTime"
    // 进入恐慌状态
    EventPanicStatus = "panicStatus"
)

var (
    events   *prometheus.GaugeVec
    hostname string
)

func init() {
    hostname, _ = os.Hostname()
    events = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "discovery_events",
    }, []string{"hostname", "cluster", "event"})
    prometheus.MustRegister(events)
}

// EventSet 事件值设置
func EventSet(event, cluster string, value float64) {
    events.WithLabelValues(hostname, cluster, event).Set(value)
}
