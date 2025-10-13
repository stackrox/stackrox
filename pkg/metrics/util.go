package metrics

import (
	"net"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/version"
)

// EmplaceCollector registers, or re-registers, the given metrics collector.
// Metrics collectors cannot be registered if an identical collector
// already exists. This function first unregisters each collector in case
// one already exists, then registers the replacement.
func EmplaceCollector(collectors ...prometheus.Collector) {
	for _, c := range collectors {
		prometheus.Unregister(c)
		prometheus.MustRegister(c)
	}
}

// CollectToSlice collects the metrics from the vector and places them in a metric slice.
func CollectToSlice(vec *prometheus.GaugeVec) ([]*dto.Metric, error) {
	metricC := make(chan prometheus.Metric)
	go func() {
		defer close(metricC)
		vec.Collect(metricC)
	}()
	errList := errorhelpers.NewErrorList("errors collecting metrics for vector")
	var metricSlice []*dto.Metric
	for metric := range metricC {
		dtoMetric := &dto.Metric{}
		errList.AddError(metric.Write(dtoMetric))
		metricSlice = append(metricSlice, dtoMetric)
	}
	return metricSlice, errList.ToError()
}

// GetBuildType returns the build type of the binary for telemetry purposes.
func GetBuildType() string {
	if version.IsReleaseVersion() {
		return "release"
	}
	return "internal"
}

func validatePort(setting env.Setting) error {
	val := setting.Setting()
	addr, err := net.ResolveTCPAddr("tcp", val)
	if err != nil {
		return err
	}
	log.Debugf("%s=%s, resolved to %+v", setting.EnvVar(), val, addr)
	return nil
}

func validateTLS() error {
	certFile := filepath.Join(env.SecureMetricsCertDir.Setting(), env.TLSCertFileName)
	if ok, err := fileutils.Exists(certFile); !ok {
		if err != nil {
			log.Errorf("failed to validate file %q: %s", certFile, err.Error())
		}
		return errors.Wrapf(errox.NotFound, "secure metrics certificate file %q not found", certFile)
	}

	keyFile := filepath.Join(env.SecureMetricsCertDir.Setting(), env.TLSKeyFileName)
	if ok, err := fileutils.Exists(keyFile); !ok {
		if err != nil {
			log.Errorf("failed to validate file %q: %s", keyFile, err.Error())
		}
		return errors.Wrapf(errox.NotFound, "secure metrics key file %q not found", keyFile)
	}
	return nil
}

// validateMetricsSetting returns an error if the environment variable is invalid.
func validateMetricsSetting() error {
	if !env.MetricsEnabled() {
		return nil
	}
	return validatePort(env.MetricsPort)
}

// validateSecureMetricsSetting returns an error if the environment variable is invalid.
func validateSecureMetricsSetting() error {
	if !env.SecureMetricsEnabled() {
		return nil
	}
	if err := validateTLS(); err != nil {
		return err
	}
	return validatePort(env.SecureMetricsPort)
}
