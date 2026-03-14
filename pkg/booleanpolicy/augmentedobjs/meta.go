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
	fileAccessKey             = "FileAccess"

	// Custom augments
	dockerfileLineAugmentKey      = "DockerfileLine"
	componentAndVersionAugmentKey = "ComponentAndVersion"
	imageSignatureVerifiedKey     = "ImageSignatureVerified"
	baselineResultAugmentKey      = "BaselineResult"
	envVarAugmentKey              = "EnvironmentVariable"
	impersonatedEventResultKey    = "ImpersonatedEventResult"
	fileAccessPathKey             = "FilePath"
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

	// This is a specialized version of DeploymentMeta for file access events.
	// FileAccessMeta has been added which includes both file access criteria and process criteria
	// (since FileAccess events contain process information).
	//
	// This enables policies to detect file access events based on both file and process criteria.
	DeploymentFileAccessMeta = pathutil.NewAugmentedObjMeta((*storage.Deployment)(nil)).
					AddAugmentedObjectAt([]string{"Containers", imageAugmentKey}, ImageMeta).
					AddPlainObjectAt([]string{"Containers", "Config", "Env", envVarAugmentKey}, (*envVar)(nil)).
					AddPlainObjectAt([]string{kubeEventAugKey}, (*storage.KubernetesEvent)(nil)).
					AddAugmentedObjectAt([]string{networkFlowAugKey}, NetworkFlowMeta).
					AddAugmentedObjectAt([]string{networkPoliciesAppliedKey}, NetworkPoliciesAppliedMeta).
					AddAugmentedObjectAt([]string{fileAccessKey}, FileAccessMeta)

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

	FileAccessMeta = pathutil.NewAugmentedObjMeta((*storage.FileAccess)(nil)).
			AddPlainObjectAt([]string{fileAccessPathKey}, (*fileAccessPath)(nil))

	NodeMeta = pathutil.NewAugmentedObjMeta((*NodeDetails)(nil)).
			AddAugmentedObjectAt([]string{fileAccessKey}, FileAccessMeta)
)
