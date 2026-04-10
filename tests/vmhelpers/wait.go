package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrAuthenticationExpired is returned when an API call fails with an
// authentication/authorization error that typically indicates an expired
// kubeconfig token or revoked credentials. Tests should stop immediately
// when this is encountered rather than retrying.
var ErrAuthenticationExpired = errors.New("authentication expired — kubeconfig token or API credentials may have expired; remaining operations will fail")

// IsAuthenticationExpired reports whether err looks like an expired or revoked
// credential. It checks gRPC Unauthenticated status codes and Kubernetes
// "Unauthorized" API errors.
func IsAuthenticationExpired(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrAuthenticationExpired) {
		return true
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.Unauthenticated {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Unauthorized") || strings.Contains(msg, "the server has asked for the client to provide credentials")
}

// WaitOptions configures a single-condition poll loop used by Central wait helpers.
type WaitOptions struct {
	Timeout      time.Duration
	PollInterval time.Duration
	// Logf, when set, is called on each unsuccessful poll with the condition
	// description and current detail so operators can follow progress in real time.
	Logf func(string, ...any)
}

// validateWaitOptions returns an error if Timeout or PollInterval are non-positive.
func validateWaitOptions(desc string, opts WaitOptions) error {
	if opts.Timeout <= 0 {
		return fmt.Errorf("vmhelpers: %s: WaitOptions.Timeout must be positive", desc)
	}
	if opts.PollInterval <= 0 {
		return fmt.Errorf("vmhelpers: %s: WaitOptions.PollInterval must be positive", desc)
	}
	return nil
}

// pollUntil runs poll until it returns done==true or ctx deadline/opts.Timeout elapses.
// detail is included in timeout errors for targeted diagnostics.
func pollUntil(ctx context.Context, opts WaitOptions, desc string, poll func(ctx context.Context) (done bool, detail string, err error)) error {
	if err := validateWaitOptions(desc, opts); err != nil {
		return err
	}
	deadline := time.Now().Add(opts.Timeout)
	var lastDetail string
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("vmhelpers: %s: %w", desc, err)
		}
		done, detail, err := poll(ctx)
		if detail != "" {
			lastDetail = detail
		}
		if err != nil {
			if IsAuthenticationExpired(err) {
				return fmt.Errorf("vmhelpers: %s: %w: %v", desc, ErrAuthenticationExpired, err)
			}
			return fmt.Errorf("vmhelpers: %s: %w", desc, err)
		}
		if done {
			if opts.Logf != nil && detail != "" {
				opts.Logf("poll %s: done (%s)", desc, detail)
			}
			return nil
		}
		if opts.Logf != nil && detail != "" {
			opts.Logf("poll %s: waiting (%s)", desc, detail)
		}
		if time.Now().After(deadline) {
			if lastDetail != "" {
				return fmt.Errorf("vmhelpers: timeout waiting for %s after %v (last detail: %s)", desc, opts.Timeout, lastDetail)
			}
			return fmt.Errorf("vmhelpers: timeout waiting for %s after %v", desc, opts.Timeout)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("vmhelpers: %s: %w", desc, ctx.Err())
		case <-time.After(opts.PollInterval):
		}
	}
}
