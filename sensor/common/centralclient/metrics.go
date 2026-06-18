package centralclient

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var proxyToCentralEnabled = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: metrics.PrometheusNamespace,
	Subsystem: metrics.SensorSubsystem.String(),
	Name:      "proxy_to_central_enabled",
	Help:      "Indicates whether Sensor connects to Central via an egress proxy. Value 1 means proxy is used.",
})

var proxyToCentralInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: metrics.PrometheusNamespace,
	Subsystem: metrics.SensorSubsystem.String(),
	Name:      "proxy_to_central_info",
	Help:      "Reports whether Sensor connects to Central through a proxy or directly via the address label.",
}, []string{"address"})

func init() {
	metrics.EmplaceCollector(proxyToCentralEnabled, proxyToCentralInfo)
}

func setProxyToCentralMetric(proxyHost string, lookupFailed bool) {
	proxyToCentralInfo.Reset()
	if lookupFailed {
		proxyToCentralEnabled.Set(0)
		proxyToCentralInfo.WithLabelValues("unknown").Set(1)
		return
	}
	if proxyHost != "" {
		proxyToCentralEnabled.Set(1)
		proxyToCentralInfo.WithLabelValues(proxyHost).Set(1)
		return
	}
	proxyToCentralEnabled.Set(0)
	proxyToCentralInfo.WithLabelValues("direct").Set(1)
}
