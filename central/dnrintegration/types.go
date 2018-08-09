package dnrintegration

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// DNRIntegration exposes all functionality that we expect to get through the integration with Detect & Respond.
type DNRIntegration interface {
	// Test tests the integration with D&R
	Test() error

	// Alerts returns D&R alerts for the given namespace and serviceName
	Alerts(namespace, serviceName string) ([]PolicyAlert, error)
}

// Validate validates a proto DNR integration object
func Validate(integration *v1.DNRIntegration) error {
	_, err := validateAndParseDirectorEndpoint(integration.GetDirectorEndpoint())
	if err != nil {
		return fmt.Errorf("directorURL invalid: %s", err)
	}

	// Some (non-comprehensive) validation of the auth token.
	// Not doing more comprehensive validation with the JWT library because
	// it seems excessive, and unnecessarily tight coupling.
	if len(integration.GetAuthToken()) < 800 {
		return fmt.Errorf("auth token too short: %d characters", len(integration.GetAuthToken()))
	}
	if !strings.HasPrefix(integration.GetAuthToken(), "ey") {
		return errors.New("auth token doesn't seem like a valid JWT (doesn't start with ey)")
	}
	return nil
}

// New returns a ready-to-use DNRIntegration object from the proto.
func New(integration *v1.DNRIntegration) (DNRIntegration, error) {
	directorURL, err := validateAndParseDirectorEndpoint(integration.GetDirectorEndpoint())
	if err != nil {
		return nil, fmt.Errorf("director URL failed validation/parsing: %s", err)
	}

	return &dnrIntegrationImpl{
		directorURL: directorURL,
		authToken:   integration.GetAuthToken(),
		client:      client,
	}, nil
}
