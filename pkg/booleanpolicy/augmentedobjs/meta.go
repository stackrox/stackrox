package augmentedobjs

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
)

const (
	imageAugmentKey           = "Image"
	processAugmentKey         = "ProcessIndicator"
	kubeEventAugKey           = "KubernetesEvent"
	networkFlowAugKey         = "NetworkFlow"
	networkPoliciesAppliedKey = "NetworkPoliciesApplied"

	// Custom augments
	dockerfileLineAugmentKey      = "DockerfileLine"
	componentAndVersionAugmentKey = "ComponentAndVersion"
	imageSignatureVerifiedKey     = "ImageSignatureVerified"
	baselineResultAugmentKey      = "BaselineResult"
	envVarAugmentKey              = "EnvironmentVariable"
	impersonatedEventResultKey    = "ImpersonatedEventResult"
)

// This block enumerates metadata about the augmented objects we use in policies.
var (
	DeploymentMeta = pathutil.NewAugmentedObjMeta((*storage.Deployment)(nil)).
			AddAugmentedObjectAt([]string{"Containers", imageAugmentKey}, ImageMeta).
			AddAugmentedObjectAt([]string{"Containers", processAugmentKey}, ProcessMeta).
			AddPlainObjectAt([]string{"Containers", "Config", "Env", envVarAugmentKey}, (*envVar)(nil)).
			AddPlainObjectAt([]string{kubeEventAugKey}, (*storage.KubernetesEvent)(nil)).
			AddAugmentedObjectAt([]string{networkFlowAugKey}, NetworkFlowMeta).
			AddAugmentedObjectAt([]string{networkPoliciesAppliedKey}, NetworkPoliciesAppliedMeta)

	ImageMeta = pathutil.NewAugmentedObjMeta((*storage.Image)(nil)).
			AddPlainObjectAt([]string{"Metadata", "V1", "Layers", dockerfileLineAugmentKey}, (*dockerfileLine)(nil)).
			AddPlainObjectAt([]string{"Scan", "Components", componentAndVersionAugmentKey}, (*componentAndVersion)(nil)).
			AddPlainObjectAt([]string{"SignatureVerificationData", imageSignatureVerifiedKey}, (*imageSignatureVerification)(nil))

	ProcessMeta = pathutil.NewAugmentedObjMeta((*storage.ProcessIndicator)(nil)).
			AddPlainObjectAt([]string{baselineResultAugmentKey}, (*baselineResult)(nil))

	KubeEventMeta = pathutil.NewAugmentedObjMeta((*storage.KubernetesEvent)(nil)).
			AddPlainObjectAt([]string{impersonatedEventResultKey}, (*impersonatedEventResult)(nil))

	NetworkFlowMeta = pathutil.NewAugmentedObjMeta((*NetworkFlowDetails)(nil))

	NetworkPoliciesAppliedMeta = pathutil.NewAugmentedObjMeta((*NetworkPoliciesApplied)(nil))
)
