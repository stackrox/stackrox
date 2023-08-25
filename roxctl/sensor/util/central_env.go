package util

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	warningKernelSupportAvailableUnimplemented = `Central does not support API for checking if kernel support is available. Not using a slim collector image.
Please upgrade Central if slim collector images shall be used.`
	errorWhenCallingGetKernelSupportAvailable = "error checking kernel support availability via GetKernelSupportAvailable Central gRPC method"
	warningLegacyCentralDefaultMain           = "Central is running on a legacy version, main image will be defaulted to %s unless overridden by user."
	warningNoClusterDefaultsAPI               = "Central does not implement GetClusterDefaultValues gRPC method, this is likely an older Central version."
	errorWhenCallingGetClusterDefaults        = "error obtaining default cluster settings from GetClusterDefaultValues Central gRPC method"
)

// CentralEnv contains information about Central's runtime environment.
type CentralEnv struct {
	// KernelSupportAvailable will be fetched from ClusterDefaults API. If talking to a legacy central, the data will
	// be fetched from legacy API GetKernelSupportAvailable. If both calls fail, false is returned and the error is
	// stored in Error property.
	KernelSupportAvailable bool
	// MainImage will only be set if central most likely won't accept an empty main image. In this case roxctl
	// will try to derive the default MainImage locally by checking the release flag.
	// If connecting to a newer (>= 3.69) version of central MainImage will be empty.
	MainImage string
	Warnings  []string
}

func getMainImageFromBuildTimeImageFlavor() string {
	var flavor defaults.ImageFlavor
	if buildinfo.ReleaseBuild {
		flavor = defaults.RHACSReleaseImageFlavor()
	} else {
		flavor = defaults.DevelopmentBuildImageFlavor()
	}
	return flavor.MainImageNoTag()
}

// RetrieveCentralEnvOrDefault populates a `CentralEnv` struct with defaults and warnings. This function fallbacks to
// legacy APIs if the defaults are not available. Ultimately, it will default values locally when it can't determine
// configuration from central. Warnings are to be used by the caller to properly display them.
// If there is an error fetching defaults from APIs, there WILL be some non-nil *CentralEnv with fallback defaults AND
// the error.
func RetrieveCentralEnvOrDefault(ctx context.Context, service v1.ClustersServiceClient) (*CentralEnv, error) {
	// These are defaults to return in case none of APIs used here works or central is too old.
	env := CentralEnv{
		MainImage:              getMainImageFromBuildTimeImageFlavor(),
		KernelSupportAvailable: false,
	}

	clusterDefaults, err := service.GetClusterDefaultValues(ctx, &v1.Empty{})
	if err == nil {
		env.KernelSupportAvailable = clusterDefaults.GetKernelSupportAvailable()
		env.MainImage = clusterDefaults.GetMainImageRepository()
		return &env, nil
	} else if status.Convert(err).Code() == codes.Unimplemented {
		env.Warnings = append(env.Warnings, warningNoClusterDefaultsAPI)
	} else {
		return &env, errors.Wrap(err, errorWhenCallingGetClusterDefaults)
	}

	// At this point we know that we're talking to older Central, therefore we tell that we're using MainImage from the
	// build-time flavor of roxctl. This might be an issue for folks who deployed older Central with `stackrox.io` image
	// flavor because roxctl will pass that default, and the generated bundle will have image references either from
	// `rhacs` flavor for the release build or a development one, but not `stackrox.io`.
	env.Warnings = append(env.Warnings, fmt.Sprintf(warningLegacyCentralDefaultMain, env.MainImage))

	// Use legacy API to determine if Central has access to kernel drivers.
	resp, err := service.GetKernelSupportAvailable(ctx, &v1.Empty{})
	if err == nil {
		env.KernelSupportAvailable = resp.GetKernelSupportAvailable()
		return &env, nil
	} else if status.Convert(err).Code() == codes.Unimplemented {
		env.Warnings = append(env.Warnings, warningKernelSupportAvailableUnimplemented)
	} else {
		return &env, errors.Wrap(err, errorWhenCallingGetKernelSupportAvailable)
	}

	return &env, nil
}
