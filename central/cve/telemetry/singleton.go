package telemetry

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once        sync.Once
	instance    *vulnerabilityMetricsImpl
	instanceMux sync.RWMutex

	log           = logging.LoggerForModule()
	Problemetrics = prometheus.NewRegistry()
)

func Singleton() interface {
	Start()
	Stop()
} {
	once.Do(func() {
		systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(
			sac.WithAllAccess(context.Background()))
		if err != nil {
			log.Errorw("Failed to get Prometheus metrics configuration", logging.Err(err))
			return
		}
		_ = ReloadConfig(systemPrivateConfig.GetPrometheusMetricsConfig())
	})
	instanceMux.RLock()
	defer instanceMux.RUnlock()
	return instance
}

func ReloadConfig(cfg *storage.PrometheusMetricsConfig) error {
	metricsConfig, period, err := parseConfig(cfg)
	if err != nil {
		log.Errorw("Failed to parse Prometheus metrics configuration", logging.Err(err))
		return err
	}
	if period == 0 {
		log.Info("No configured Prometheus metrics")
	}
	instanceMux.Lock()
	defer instanceMux.Unlock()
	if instance != nil {
		instance.metricsConfig = metricsConfig
		instance.period = period
		instance.periodCh <- period
	} else {
		instance = &vulnerabilityMetricsImpl{
			aggregator: &aggregator{
				ds:        deploymentDS.Singleton(),
				trackFunc: metrics.SetAggregatedVulnCount,
			},
			metricsConfig: metricsConfig,
			period:        period,
			periodCh:      make(chan time.Duration),
		}
	}
	instance.registerMetrics(log)
	return nil
}

func (impl *vulnerabilityMetricsImpl) registerMetrics(log logging.Logger) {
	for metric, expressions := range impl.metricsConfig {
		metrics.RegisterVulnAggregatedMetric(string(metric), impl.period,
			getMetricLabels(expressions), Problemetrics)

		log.Infof("Registered Prometheus metric %q", metric)
	}
}

type metricName string
type metricKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true
type metricsConfig map[metricName]map[Label][]*expression

type vulnerabilityMetricsImpl struct {
	metricsConfig metricsConfig

	aggregator *aggregator

	stopSignal chan bool
	period     time.Duration
	periodCh   chan time.Duration
}

func (impl *vulnerabilityMetricsImpl) Start() {
	if impl != nil {
		go impl.run()
	}
}

func (impl *vulnerabilityMetricsImpl) Stop() {
	if impl != nil {
		close(impl.stopSignal)
	}
}

func (impl *vulnerabilityMetricsImpl) getMetricsConfig() metricsConfig {
	instanceMux.RLock()
	defer instanceMux.RUnlock()
	return impl.metricsConfig
}

func (impl *vulnerabilityMetricsImpl) run() {
	ticker := time.NewTicker(impl.period)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(
		sac.WithAllAccess(context.Background()))

	impl.aggregator.track(ctx, impl.getMetricsConfig())
	defer cancel()
	for {
		select {
		case <-ticker.C:
			impl.aggregator.track(ctx, impl.getMetricsConfig())
		case <-impl.stopSignal:
			return
		case period := <-impl.periodCh:
			if period > 0 {
				ticker.Reset(period)
			} else {
				ticker.Stop()
			}
		}
	}
}
