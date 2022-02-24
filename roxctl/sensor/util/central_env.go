package util

import (
	"context"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	warningKernelSupportAvailableUnimplemented = `Central does not support API for checking if kernel support is available. Not using a slim collector image.
Please upgrade Central if slim collector images shall be used.`
	warningFailedAToGetKernelSupportAvailable = "Failed to retrieve KernelSupportAvailable property from Central"
	warningCantDetermineMainImageFromCentral  = "Can't rely on central configuration to determine default values. Using %s as main registry."
)

// CentralEnv contains information about Central's runtime environment.
type CentralEnv struct {
	KernelSupportAvailable bool
	MainImage              string
	Warnings               []string
	Error                  error
}

func getFlavorFromReleaseBuild() defaults.ImageFlavor {
	if buildinfo.ReleaseBuild {
		return defaults.RHACSReleaseImageFlavor()
	}
	return defaults.DevelopmentBuildImageFlavor()
}

func (e *CentralEnv) fetchClusterDefaults(ctx context.Context, service v1.ClustersServiceClient) error {
	clusterDefault, err := service.GetClusterDefaults(ctx, &v1.Empty{})
	if err != nil {
		return err
	}
	e.KernelSupportAvailable = clusterDefault.GetKernelSupportAvailable()
	e.MainImage = clusterDefault.GetMainImageRepository()
	return nil
}

func (e *CentralEnv) fetchClusterDefaultsLegacy(ctx context.Context, service v1.ClustersServiceClient) {
	resp, err := service.GetKernelSupportAvailable(ctx, &v1.Empty{})
	if err != nil {
		if status.Convert(err).Code() == codes.Unimplemented {
			e.Warnings = append(e.Warnings, warningKernelSupportAvailableUnimplemented)
		} else {
			e.Warnings = append(e.Warnings, warningFailedAToGetKernelSupportAvailable)
		}
		// If all APIs fail, we default KernelSupportAvailable to false and store the error
		e.KernelSupportAvailable = false
		e.Error = err
	} else {
		e.KernelSupportAvailable = resp.KernelSupportAvailable
	}

	flavor := getFlavorFromReleaseBuild()
	e.MainImage = flavor.MainImageNoTag()
	e.Warnings = append(e.Warnings, fmt.Sprintf(warningCantDetermineMainImageFromCentral, e.MainImage))
}

func (e *CentralEnv) populateCentralEnv(ctx context.Context, service v1.ClustersServiceClient) {
	err := e.fetchClusterDefaults(ctx, service)
	if err != nil {
		e.fetchClusterDefaultsLegacy(ctx, service)
	}
}

// RetrieveCentralEnvOrDefault is a convenience function wrapping `PopulateCentralEnv`. It populates a fresh `CentralEnv`
// struct with defaults and warnings. Warnings are to be used by the caller to properly display them.
// If there is an error fetching defaults from the API the error will be returned in `CentralEnv`.
func RetrieveCentralEnvOrDefault(ctx context.Context, service v1.ClustersServiceClient) CentralEnv {
	env := CentralEnv{}
	env.populateCentralEnv(ctx, service)
	return env
}
