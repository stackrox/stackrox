package augmentedobjs

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
)

const (
	imageAugmentKey   = "Image"
	processAugmentKey = "ProcessIndicator"

	// Custom augments
	dockerfileLineAugmentKey      = "DockerfileLine"
	componentAndVersionAugmentKey = "ComponentAndVersion"
	whitelistResultAugmentKey     = "WhitelistResult"
	envVarAugmentKey              = "EnvironmentVariable"
)

// This block enumerates metadata about the augmented objects we use in policies.
var (
	DeploymentMeta = pathutil.NewAugmentedObjMeta((*storage.Deployment)(nil)).
			AddAugmentedObjectAt([]string{"Containers", imageAugmentKey}, ImageMeta).
			AddAugmentedObjectAt(
			[]string{"Containers", processAugmentKey},
			pathutil.NewAugmentedObjMeta((*storage.ProcessIndicator)(nil)).
				AddPlainObjectAt([]string{whitelistResultAugmentKey}, (*whitelistResult)(nil)),
		).AddPlainObjectAt([]string{"Containers", "Config", "Env", envVarAugmentKey}, (*envVar)(nil))

	ImageMeta = pathutil.NewAugmentedObjMeta((*storage.Image)(nil)).
			AddPlainObjectAt([]string{"Metadata", "V1", "Layers", dockerfileLineAugmentKey}, (*dockerfileLine)(nil)).
			AddPlainObjectAt([]string{"Scan", "Components", componentAndVersionAugmentKey}, (*componentAndVersion)(nil))
)
