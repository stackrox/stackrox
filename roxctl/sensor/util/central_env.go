package util

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	warningFailedToGetKernelSupportAvailable = `Central does not support API for checking if kernel support is available. Not using a slim collector image.
Please upgrade Central if slim collector images shall be used.`
)

// CentralEnv contains information about Central's runtime environment.
type CentralEnv struct {
	KernelSupportAvailable bool
}

// DefaultCentralEnv creates a new `CentralEnv` struct populated with default values..
func DefaultCentralEnv() *CentralEnv {
	return &CentralEnv{}
}

func (e *CentralEnv) populateCentralEnv(ctx context.Context, service v1.ClustersServiceClient) error {
	resp, err := service.GetKernelSupportAvailable(ctx, &v1.Empty{})
	if err != nil {
		if status.Convert(err).Code() != codes.Unimplemented {
			return errors.Wrap(err, "failed to retrieve KernelSupportAvailable property from Central")
		}
		fmt.Fprintln(os.Stderr, warningFailedToGetKernelSupportAvailable)
		resp = &v1.KernelSupportAvailableResponse{KernelSupportAvailable: false} // optional but makes the intention clearer
	}
	e.KernelSupportAvailable = resp.GetKernelSupportAvailable()

	return nil
}

// RetrieveCentralEnvOrDefault is a convenience function wrapping `PopulateCentralEnv`. It populates a fresh `CentralEnv`
// struct and in case this fails with an error, the error is swallowed and an informative error message is printed to stderr.
// In any case the caller receives a `CentralEnv`.
func RetrieveCentralEnvOrDefault(ctx context.Context, service v1.ClustersServiceClient) *CentralEnv {
	env := DefaultCentralEnv()

	err := env.populateCentralEnv(ctx, service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query Central for its environment: %v. Assuming default environment.", err)
	}

	return env
}
