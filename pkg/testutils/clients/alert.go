package clients

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

type alertClient struct {
	conn    *grpc.ClientConn
	service v1.AlertServiceClient
}

func (a *alertClient) GetAlert(ctx context.Context, alertID string) (*storage.Alert, error) {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	resp, err := a.service.GetAlert(ctx, &v1.ResourceByID{Id: alertID})
	if err != nil {
		return nil, fmt.Errorf("failed to get alert %s: %w", alertID, err)
	}

	return resp, nil
}

func (a *alertClient) ListAlerts(ctx context.Context, query string) ([]*storage.ListAlert, error) {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	resp, err := a.service.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts with query '%s': %w", query, err)
	}

	return resp.GetAlerts(), nil
}

func (a *alertClient) GetAlertsForPolicy(ctx context.Context, policyID string) ([]*storage.Alert, error) {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	// Build query to find alerts for specific policy
	query := search.NewQueryBuilder().
		AddExactMatches(search.PolicyID, policyID).
		Query()

	listResp, err := a.service.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts for policy %s: %w", policyID, err)
	}

	// Fetch full alert details
	var alerts []*storage.Alert
	for _, listAlert := range listResp.GetAlerts() {
		alert, err := a.GetAlert(ctx, listAlert.GetId())
		if err != nil {
			return nil, fmt.Errorf("failed to get alert details for %s: %w", listAlert.GetId(), err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (a *alertClient) GetAlertsForDeployment(ctx context.Context, deploymentID string) ([]*storage.Alert, error) {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	query := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, deploymentID).
		Query()

	listResp, err := a.service.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts for deployment %s: %w", deploymentID, err)
	}

	// Fetch full alert details
	var alerts []*storage.Alert
	for _, listAlert := range listResp.GetAlerts() {
		alert, err := a.GetAlert(ctx, listAlert.GetId())
		if err != nil {
			return nil, fmt.Errorf("failed to get alert details for %s: %w", listAlert.GetId(), err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (a *alertClient) GetAlertsForImage(ctx context.Context, imageName string) ([]*storage.Alert, error) {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	query := search.NewQueryBuilder().
		AddExactMatches(search.ImageName, imageName).
		Query()

	listResp, err := a.service.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts for image %s: %w", imageName, err)
	}

	// Fetch full alert details
	var alerts []*storage.Alert
	for _, listAlert := range listResp.GetAlerts() {
		alert, err := a.GetAlert(ctx, listAlert.GetId())
		if err != nil {
			return nil, fmt.Errorf("failed to get alert details for %s: %w", listAlert.GetId(), err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (a *alertClient) ResolveAlert(ctx context.Context, alertID string) error {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	_, err := a.service.ResolveAlert(ctx, &v1.ResolveAlertRequest{
		Id: alertID,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve alert %s: %w", alertID, err)
	}

	return nil
}

func (a *alertClient) DeleteAlert(ctx context.Context, alertID string) error {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	_, err := a.service.DeleteAlerts(ctx, &v1.DeleteAlertsRequest{
		Query: search.NewQueryBuilder().AddExactMatches(search.AlertID, alertID).Query(),
	})
	if err != nil {
		return fmt.Errorf("failed to delete alert %s: %w", alertID, err)
	}

	return nil
}

// AlertWaitOptions configures how to wait for alerts
type AlertWaitOptions struct {
	Timeout         time.Duration
	CheckInterval   time.Duration
	MinAlertCount   int
	MaxAlertCount   int
	ExpectedSeverity *storage.Severity
}

// DefaultAlertWaitOptions provides sensible defaults for waiting for alerts
func DefaultAlertWaitOptions() *AlertWaitOptions {
	return &AlertWaitOptions{
		Timeout:       2 * time.Minute,
		CheckInterval: 5 * time.Second,
		MinAlertCount: 1,
		MaxAlertCount: -1, // No upper limit
	}
}

// WaitForPolicyAlerts waits for alerts to be generated for a specific policy
func (a *alertClient) WaitForPolicyAlerts(ctx context.Context, policyID string, opts *AlertWaitOptions) ([]*storage.Alert, error) {
	if opts == nil {
		opts = DefaultAlertWaitOptions()
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	ticker := time.NewTicker(opts.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for alerts for policy %s: %w", policyID, ctx.Err())
		case <-ticker.C:
			alerts, err := a.GetAlertsForPolicy(ctx, policyID)
			if err != nil {
				continue // Retry on error
			}

			// Check if we have enough alerts
			if len(alerts) >= opts.MinAlertCount {
				// Check max count constraint
				if opts.MaxAlertCount > 0 && len(alerts) > opts.MaxAlertCount {
					continue // Too many alerts, keep waiting
				}

				// Check severity constraint if specified
				if opts.ExpectedSeverity != nil {
					validAlerts := a.filterAlertsBySeverity(alerts, *opts.ExpectedSeverity)
					if len(validAlerts) >= opts.MinAlertCount {
						return validAlerts, nil
					}
					continue // Not enough alerts with correct severity
				}

				return alerts, nil
			}
		}
	}
}

// WaitForDeploymentAlerts waits for alerts to be generated for a specific deployment
func (a *alertClient) WaitForDeploymentAlerts(ctx context.Context, deploymentID string, opts *AlertWaitOptions) ([]*storage.Alert, error) {
	if opts == nil {
		opts = DefaultAlertWaitOptions()
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	ticker := time.NewTicker(opts.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for alerts for deployment %s: %w", deploymentID, ctx.Err())
		case <-ticker.C:
			alerts, err := a.GetAlertsForDeployment(ctx, deploymentID)
			if err != nil {
				continue // Retry on error
			}

			if len(alerts) >= opts.MinAlertCount {
				if opts.MaxAlertCount > 0 && len(alerts) > opts.MaxAlertCount {
					continue
				}

				if opts.ExpectedSeverity != nil {
					validAlerts := a.filterAlertsBySeverity(alerts, *opts.ExpectedSeverity)
					if len(validAlerts) >= opts.MinAlertCount {
						return validAlerts, nil
					}
					continue
				}

				return alerts, nil
			}
		}
	}
}

// filterAlertsBySeverity filters alerts by severity level
func (a *alertClient) filterAlertsBySeverity(alerts []*storage.Alert, severity storage.Severity) []*storage.Alert {
	var filtered []*storage.Alert
	for _, alert := range alerts {
		if alert.GetPolicy().GetSeverity() == severity {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

// GetRecentAlerts returns alerts created within the specified duration
func (a *alertClient) GetRecentAlerts(ctx context.Context, since time.Duration) ([]*storage.Alert, error) {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	// Calculate the time threshold
	threshold := time.Now().Add(-since)

	query := search.NewQueryBuilder().
		AddTimeRangeField(search.ViolationTime, threshold, time.Now()).
		Query()

	listResp, err := a.service.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list recent alerts: %w", err)
	}

	// Fetch full alert details
	var alerts []*storage.Alert
	for _, listAlert := range listResp.GetAlerts() {
		alert, err := a.GetAlert(ctx, listAlert.GetId())
		if err != nil {
			return nil, fmt.Errorf("failed to get alert details for %s: %w", listAlert.GetId(), err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// DeleteAllAlerts deletes all alerts matching the given query
func (a *alertClient) DeleteAllAlerts(ctx context.Context, query string) error {
	if a.service == nil {
		a.service = v1.NewAlertServiceClient(a.conn)
	}

	_, err := a.service.DeleteAlerts(ctx, &v1.DeleteAlertsRequest{
		Query: query,
	})
	if err != nil {
		return fmt.Errorf("failed to delete alerts with query '%s': %w", query, err)
	}

	return nil
}