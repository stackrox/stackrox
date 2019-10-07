package env

import "net"

const (
	defaultMetricsSetting = ":9090"
)

var (
	// MetricsSetting has the :port or host:port string for listening for metrics/debug server
	MetricsSetting = RegisterSetting("ROX_METRICS_PORT", WithDefault(defaultMetricsSetting))
)

// ValidateMetricsSetting returns an error if the environment variable is invalid.
func ValidateMetricsSetting() error {
	val := MetricsSetting.Setting()
	if val == "disabled" {
		return nil
	}
	addr, err := net.ResolveTCPAddr("tcp", val)
	if err != nil {
		return err
	}
	log.Debugf("%s=%s, resolved to %+v", MetricsSetting.EnvVar(), val, addr)
	return nil
}

// MetricsEnabled returns true if the metrics/debug http server should be started
func MetricsEnabled() bool {
	return MetricsSetting.Setting() != "disabled"
}
