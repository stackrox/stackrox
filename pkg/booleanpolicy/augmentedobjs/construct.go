package augmentedobjs

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// CompositeFieldCharSep is the separating character used when we create a composite field.
	CompositeFieldCharSep = "\t"
)

func findMatchingContainerIdxForProcess(deployment *storage.Deployment, process *storage.ProcessIndicator) (int, error) {
	for i, container := range deployment.GetContainers() {
		if container.GetName() == process.GetContainerName() {
			return i, nil
		}
	}
	return 0, errors.Errorf("indicator %s could not be matched (container name %s not found in deployment %s/%s/%s",
		process.GetSignal().GetExecFilePath(), process.GetContainerName(), deployment.GetClusterId(), deployment.GetNamespace(), deployment.GetName())

}

// ConstructDeploymentWithProcess constructs an augmented deployment with process information.
func ConstructDeploymentWithProcess(deployment *storage.Deployment, images []*storage.Image, applied *NetworkPoliciesApplied, process *storage.ProcessIndicator, processNotInBaseline bool) (*pathutil.AugmentedObj, error) {
	obj, err := ConstructDeployment(deployment, images, applied)
	if err != nil {
		return nil, err
	}
	augmentedProcess, err := ConstructProcess(process, processNotInBaseline)
	if err != nil {
		return nil, err
	}

	matchingContainerIdx, err := findMatchingContainerIdxForProcess(deployment, process)
	if err != nil {
		return nil, err
	}
	err = obj.AddAugmentedObjAt(
		augmentedProcess,
		pathutil.FieldStep("Containers"), pathutil.IndexStep(matchingContainerIdx), pathutil.FieldStep(processAugmentKey),
	)
	if err != nil {
		return nil, utils.ShouldErr(err)
	}
	return obj, nil
}

// ConstructKubeResourceWithEvent constructs an augmented deployment with kube event information.
func ConstructKubeResourceWithEvent(kubeResource interface{}, event *storage.KubernetesEvent) (*pathutil.AugmentedObj, error) {
	obj, isAugmented := kubeResource.(*pathutil.AugmentedObj)
	if !isAugmented {
		if !supportedKubeResourceForEvent(kubeResource) {
			return nil, errors.Errorf("unsupported kubernetes resource type %T for event detection", kubeResource)
		}
		obj = pathutil.NewAugmentedObj(kubeResource)
	}

	if err := obj.AddPlainObjAt(event, pathutil.FieldStep(kubeEventAugKey)); err != nil {
		return nil, utils.ShouldErr(err)
	}
	return obj, nil
}

func supportedKubeResourceForEvent(obj interface{}) bool {
	switch obj.(type) {
	case *storage.Deployment:
		return true
	default:
		return false
	}
}

// ConstructProcess constructs an augmented process.
func ConstructProcess(process *storage.ProcessIndicator, processNotInBaseline bool) (*pathutil.AugmentedObj, error) {
	augmentedProcess := pathutil.NewAugmentedObj(process)
	err := augmentedProcess.AddPlainObjAt(
		&baselineResult{NotInBaseline: processNotInBaseline},
		pathutil.FieldStep(baselineResultAugmentKey),
	)
	if err != nil {
		return nil, errors.Wrap(err, "adding process baseline result to process")
	}
	return augmentedProcess, nil
}

// ConstructKubeEvent constructs an augmented kubernetes event.
func ConstructKubeEvent(event *storage.KubernetesEvent) *pathutil.AugmentedObj {
	return pathutil.NewAugmentedObj(event)
}

// ConstructAuditEvent constructs an augmented kubernetes event.
func ConstructAuditEvent(event *storage.KubernetesEvent, isImpersonated bool) (*pathutil.AugmentedObj, error) {
	augmentedProcess := pathutil.NewAugmentedObj(event)

	err := augmentedProcess.AddPlainObjAt(
		&impersonatedEventResult{IsImpersonatedUser: isImpersonated},
		pathutil.FieldStep(impersonatedEventResultKey),
	)
	if err != nil {
		return nil, errors.Wrap(err, "adding is impersonated user result to audit event")
	}
	return augmentedProcess, nil
}

// ConstructNetworkFlow constructs an augmented network flow.
func ConstructNetworkFlow(flow *NetworkFlowDetails) (*pathutil.AugmentedObj, error) {
	augmentedFlow := pathutil.NewAugmentedObj(flow)
	return augmentedFlow, nil
}

// ConstructDeploymentWithNetworkFlowInfo constructs an augmented object with deployment and network flow.
func ConstructDeploymentWithNetworkFlowInfo(
	deployment *storage.Deployment,
	images []*storage.Image,
	applied *NetworkPoliciesApplied,
	flow *NetworkFlowDetails,
) (*pathutil.AugmentedObj, error) {
	obj, err := ConstructDeployment(deployment, images, applied)
	if err != nil {
		return nil, err
	}
	augmentedFlow, err := ConstructNetworkFlow(flow)
	if err != nil {
		return nil, err
	}

	err = obj.AddAugmentedObjAt(augmentedFlow, pathutil.FieldStep(networkFlowAugKey))
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// ConstructDeployment constructs the augmented deployment object.
// It assumes that the given images are in the same order as the containers specified within the given deployment.
// If there's a mismatch in the amount of containers on the deployment and the given images, an error will be returned.
func ConstructDeployment(deployment *storage.Deployment, images []*storage.Image, applied *NetworkPoliciesApplied) (*pathutil.AugmentedObj, error) {
	obj := pathutil.NewAugmentedObj(deployment)
	if len(images) != len(deployment.GetContainers()) {
		return nil, errors.Errorf("deployment %s/%s had %d containers, but got %d images",
			deployment.GetNamespace(), deployment.GetName(), len(deployment.GetContainers()), len(images))
	}

	appliedPolicies := pathutil.NewAugmentedObj(applied)
	if err := obj.AddAugmentedObjAt(appliedPolicies, pathutil.FieldStep(networkPoliciesAppliedKey)); err != nil {
		return nil, utils.ShouldErr(err)
	}

	for i, image := range images {
		// Since we ensure that both images and containers have the same length, this will not lead to index out of
		// bounds panics.
		containerImageFullName := deployment.GetContainers()[i].GetImage().GetName().GetFullName()
		augmentedImg, err := ConstructImage(image, containerImageFullName)
		if err != nil {
			return nil, err
		}
		err = obj.AddAugmentedObjAt(
			augmentedImg,
			pathutil.FieldStep("Containers"), pathutil.IndexStep(i), pathutil.FieldStep(imageAugmentKey),
		)
		if err != nil {
			return nil, utils.ShouldErr(err)
		}
	}

	for idx, container := range deployment.GetContainers() {
		for i, env := range container.GetConfig().GetEnv() {
			envVarObj := &envVar{EnvVar: fmt.Sprintf("%s%s%s%s%s", env.GetEnvVarSource(), CompositeFieldCharSep, env.GetKey(), CompositeFieldCharSep, env.GetValue())}
			err := obj.AddPlainObjAt(
				envVarObj,
				pathutil.FieldStep("Containers"), pathutil.IndexStep(idx), pathutil.FieldStep("Config"),
				pathutil.FieldStep("Env"), pathutil.IndexStep(i), pathutil.FieldStep(envVarAugmentKey),
			)

			if err != nil {
				return nil, utils.ShouldErr(err)
			}
		}
	}

	return obj, nil
}

// ConstructImage constructs the augmented image object.
func ConstructImage(image *storage.Image, imageFullName string) (*pathutil.AugmentedObj, error) {
	if image == nil {
		return pathutil.NewAugmentedObj(image), nil
	}

	img := *image

	// When evaluating policies, the evaluator will stop when any of the objects within the path
	// are nil and immediately return, not matching. Within the image signature criteria, we have
	// a combination of "Match if the signature verification result is not as expected" OR "Match if
	// there is no signature verification result". This means that we have to add the SignatureVerificationData
	// and SignatureVerificationResults object here as a workaround and add the placeholder value,
	// making it possible to also match for nil objects.
	// We have to do this at the beginning, so the augmented object contains the field steps.
	if img.GetSignatureVerificationData().GetResults() == nil {
		img.SignatureVerificationData = &storage.ImageSignatureVerificationData{
			Results: []*storage.ImageSignatureVerificationResult{{}},
		}
	}

	obj := pathutil.NewAugmentedObj(&img)

	// Since policies query for Dockerfile Line as a single compound field, we simulate it by creating a "composite"
	// dockerfile line under each layer.
	for i, layer := range image.GetMetadata().GetV1().GetLayers() {
		lineObj := &dockerfileLine{Line: fmt.Sprintf("%s%s%s", layer.GetInstruction(), CompositeFieldCharSep, layer.GetValue())}
		err := obj.AddPlainObjAt(
			lineObj,
			pathutil.FieldStep("Metadata"), pathutil.FieldStep("V1"), pathutil.FieldStep("Layers"),
			pathutil.IndexStep(i), pathutil.FieldStep(dockerfileLineAugmentKey),
		)
		if err != nil {
			return nil, utils.ShouldErr(err)
		}
	}

	// Since policies query for component and version as a single compound field, we simulate it by creating a
	// "composite" component and version field.
	for i, component := range image.GetScan().GetComponents() {
		compAndVersionObj := &componentAndVersion{
			ComponentAndVersion: fmt.Sprintf("%s%s%s", component.GetName(), CompositeFieldCharSep, component.GetVersion()),
		}
		err := obj.AddPlainObjAt(
			compAndVersionObj,
			pathutil.FieldStep("Scan"), pathutil.FieldStep("Components"), pathutil.IndexStep(i),
			pathutil.FieldStep(componentAndVersionAugmentKey),
		)
		if err != nil {
			return nil, utils.ShouldErr(err)
		}
	}

	ids := []string{}
	for _, result := range image.GetSignatureVerificationData().GetResults() {
		// We only want signature verification results to be added that:
		// - have a verified result.
		// - the verified image references contains the image full name that is currently specified. This can either
		// be equal to `img.GetName().GetFullName()`, or when used within a deployment, the container image's full name.
		if result.GetStatus() == storage.ImageSignatureVerificationResult_VERIFIED &&
			sliceutils.Find(result.GetVerifiedImageReferences(), imageFullName) != -1 {
			ids = append(ids, result.GetVerifierId())
		}
	}
	// When the object is not created, the policy will not match, but it should match.
	if err := obj.AddPlainObjAt(
		&imageSignatureVerification{
			VerifierIDs: ids,
		},
		pathutil.FieldStep("SignatureVerificationData"),
		pathutil.FieldStep(imageSignatureVerifiedKey)); err != nil {
		return nil, utils.ShouldErr(err)
	}

	return obj, nil
}
