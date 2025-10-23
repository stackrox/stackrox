package ml

import (
	"sync"
	"time"

	"github.com/stackrox/rox/pkg/env"
)

var (
	// Environment variables for ML service configuration
	mlServiceEndpoint = env.RegisterSetting("ROX_ML_RISK_SERVICE_ENDPOINT", env.WithDefault("ml-risk-service:8080"))
	mlServiceEnabled  = env.RegisterBooleanSetting("ROX_ML_RISK_SERVICE_ENABLED", false)
	mlServiceTLS      = env.RegisterBooleanSetting("ROX_ML_RISK_SERVICE_TLS", false)
	mlServiceTimeout  = env.RegisterSetting("ROX_ML_RISK_SERVICE_TIMEOUT", env.WithDefault("30s"))
)

var (
	clientInstance MLRiskClient
	clientOnce     sync.Once
	clientErr      error
)

// Singleton returns the singleton ML Risk Client instance
func Singleton() MLRiskClient {
	clientOnce.Do(func() {
		if !mlServiceEnabled.BooleanSetting() {
			log.Info("ML Risk Service is disabled")
			clientInstance = &noOpClient{}
			return
		}

		timeout, timeoutErr := time.ParseDuration(mlServiceTimeout.Setting())
		if timeoutErr != nil {
			log.Errorf("Invalid timeout duration '%s', using default 30s: %v", mlServiceTimeout.Setting(), timeoutErr)
			timeout = 30 * time.Second
		}

		config := &Config{
			Endpoint:   mlServiceEndpoint.Setting(),
			TLSEnabled: mlServiceTLS.BooleanSetting(),
			Timeout:    timeout,
		}

		log.Infof("Initializing ML Risk Service client with endpoint: %s", config.Endpoint)

		client, err := NewMLRiskClient(config)
		if err != nil {
			log.Errorf("Failed to create ML Risk Service client: %v", err)
			clientErr = err
			// Fall back to no-op client
			clientInstance = &noOpClient{}
			return
		}

		clientInstance = client
		log.Info("ML Risk Service client initialized successfully")
	})

	return clientInstance
}

// IsEnabled returns whether ML Risk Service is enabled
func IsEnabled() bool {
	return mlServiceEnabled.BooleanSetting()
}

// GetClientError returns any error that occurred during client initialization
func GetClientError() error {
	// Ensure singleton is initialized
	Singleton()
	return clientErr
}

// Reset resets the singleton (primarily for testing)
func Reset() {
	clientOnce = sync.Once{}
	if clientInstance != nil {
		_ = clientInstance.Close()
	}
	clientInstance = nil
	clientErr = nil
}
