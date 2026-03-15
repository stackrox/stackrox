package manager

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/stringutils"
	admission "k8s.io/api/admission/v1"
	"k8s.io/utils/pointer"
)

const (
	ScaleSubResource = "scale"
)

var (
	detectionCtx = deploytime.DetectionContext{
		EnforcementOnly: true,
	}
)

func (m *manager) shouldBypass(s *state, req *admission.AdmissionRequest) bool {
	if !s.activeForOperation(req.Operation) {
		log.Debugf("Not enforcing on operation, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	// Do not enforce on StackRox and system namespaces
	if req.Namespace == namespaces.StackRox || req.Namespace == m.ownNamespace || kubernetes.IsSystemNamespace(req.Namespace) {
		log.Debugf("Action affects system namespace, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	// We don't enforce on subresources other than the scale subresource.
	// Openshift console uses the scale subresource to scale deployments, and our admission controller bypasses these requests
	// without running policy detection and enforcement. However, an `oc scale` command works. The following
	// change makes the behavior of admission controller consistent across all supported ways that k8s allows
	// deployment replica scaling.
	if req.SubResource != "" && req.SubResource != ScaleSubResource {
		log.Debugf("Request is for a subresource other than the scale subresource, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	// Allow the request if it comes from a blessed user
	if s.bypassForUsers.Contains(req.UserInfo.Username) {
		log.Debugf("Request comes from privileged user %s, bypassing %s request on %s/%s [%s]", req.UserInfo.Username, req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	// Allow the request if it comes from a service account in a system namespace
	if strings.HasPrefix(req.UserInfo.Username, "system:serviceaccount:") {
		saNamespace, _ := stringutils.Split2(req.UserInfo.Username[len("system:serviceaccount:"):], ":")
		if kubernetes.IsSystemNamespace(saNamespace) {
			log.Debugf("Request comes from a system service account %s, bypassing %s request on %s/%s [%s]", req.UserInfo.Username, req.Operation, req.Namespace, req.Name, req.Kind)
			return true
		}
	}

	// Allow the request if it comes from a blessed group
	for _, group := range req.UserInfo.Groups {
		if s.bypassForGroups.Contains(group) {
			log.Debugf("Request comes from privileged group %s, bypassing %s request on %s/%s [%s]", group, req.Operation, req.Namespace, req.Name, req.Kind)
			return true
		}
	}

	return false
}

// hasOnlyUnenrichedImageAlerts returns true if all alerts in the slice are from
// policies that fire on the absence of image enrichment data (UnscannedImage,
// ImageSignatureVerifiedBy). These alerts are transient false positives caused
// by placeholder images before enrichment completes.
func hasOnlyUnenrichedImageAlerts(alerts []*storage.Alert) bool {
	for _, a := range alerts {
		if !policyfields.AlertsOnMissingEnrichment(a.GetPolicy()) {
			return false
		}
	}
	return true
}

// filterOutUnenrichedImageAlerts removes alerts from policies that fire on the
// absence of image enrichment data to remove false positives. The remaining alerts are genuine violations.
// The input slice is modified in place and should not be used afterwards.
func filterOutUnenrichedImageAlerts(alerts []*storage.Alert) []*storage.Alert {
	filteredAlerts := alerts[:0]
	for _, a := range alerts {
		if policyfields.AlertsOnMissingEnrichment(a.GetPolicy()) {
			continue
		}
		filteredAlerts = append(filteredAlerts, a)
	}
	return filteredAlerts
}

func (m *manager) evaluateAdmissionRequest(s *state, req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	log.Debugf(
		"Evaluating admission request (uid=%s, ns=%s, name=%s, op=%s, kind=%s)",
		req.UID,
		req.Namespace,
		req.Name,
		string(req.Operation),
		req.Kind.String(),
	)

	if m.shouldBypass(s, req) {
		observeAdmissionReview(reviewResultBypassed, 0)
		return pass(req.UID), nil
	}

	start := time.Now()
	log.Debugf("Not bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)

	var deployment *storage.Deployment
	if req.SubResource != "" && req.SubResource == ScaleSubResource {
		if deployment = m.deployments.GetByName(req.Namespace, req.Name); deployment == nil {
			observeAdmissionReview(reviewResultError, time.Since(start))
			return nil, errors.Errorf(
				"could not find deployment with name: %q in namespace %q for this admission review request",
				req.Name, req.Namespace)
		}
	} else {
		k8sObj, err := unmarshalK8sObject(req.Kind, req.Object.Raw)
		if err != nil {
			observeAdmissionReview(reviewResultError, time.Since(start))
			return nil, errors.Wrap(err, "could not unmarshal object from request")
		}

		deployment, err = resources.NewDeploymentFromStaticResource(k8sObj, req.Kind.Kind, s.clusterID(), s.GetClusterConfig().GetRegistryOverride())
		if err != nil {
			observeAdmissionReview(reviewResultError, time.Since(start))
			return nil, errors.Wrap(err, "could not convert Kubernetes object into StackRox deployment")
		}

		if deployment == nil {
			log.Debugf("Non-top-level object, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
			observeAdmissionReview(reviewResultBypassed, 0)
			return pass(req.UID), nil // we only enforce on top-level objects
		}
	}
	log.Debugf("Evaluating policies on %+v", deployment)

	// Check if deployment has bypass annotation
	if !s.GetClusterConfig().GetAdmissionControllerConfig().GetDisableBypass() {
		if !enforcers.ShouldEnforce(deployment.GetAnnotations()) {
			log.Warnf("deployment %s/%s of type %v was deployed without being checked due to matching bypass annotation %q",
				deployment.GetNamespace(), deployment.GetName(), req.Kind, enforcers.EnforcementBypassAnnotationKey)
			observeAdmissionReview(reviewResultBypassed, 0)
			return pass(req.UID), nil
		}
	}

	if resp, err := m.evaluateFastPath(s, req, deployment, start); resp != nil || err != nil {
		return resp, err
	}

	return m.evaluateSlowPath(s, req, deployment, start)
}

// evaluateFastPath runs deployment-spec-only policies that don't require image enrichment data.
func (m *manager) evaluateFastPath(s *state, req *admission.AdmissionRequest, deployment *storage.Deployment, start time.Time) (*admission.AdmissionResponse, error) {
	// Fast path: skip entirely if there are no policies that require deployment spec fields only.
	if len(s.fastPathDeployDetector.PolicySet().GetCompiledPolicies()) == 0 {
		return nil, nil
	}

	alerts, err := s.fastPathDeployDetector.Detect(context.Background(), detectionCtx, booleanpolicy.EnhancedDeployment{
		Deployment: deployment,
		Images:     toPlaceholderImages(deployment),
	})
	if err != nil {
		observeAdmissionReview(reviewResultError, time.Since(start))
		return nil, errors.Wrapf(err, "running %s detection", branding.GetProductNameShort())
	}
	if len(alerts) == 0 {
		return nil, nil
	}

	if !pointer.BoolDeref(req.DryRun, false) {
		go m.filterAndPutAttemptedAlertsOnChan(req.Operation, alerts...)
	}
	unevaluatedPolicyCount := len(s.slowPathDeployDetector.PolicySet().GetCompiledPolicies())
	log.Debugf("Violated policies (fast path): %d, rejecting %s request on %s/%s [%s]", len(alerts), req.Operation, req.Namespace, req.Name, req.Kind)
	observeAdmissionReview(reviewResultDenied, time.Since(start))
	return fail(req.UID, message(alerts, !s.GetClusterConfig().GetAdmissionControllerConfig().GetDisableBypass(), unevaluatedPolicyCount)), nil
}

// evaluateSlowPath runs image-dependent policies, fetching and scanning images as needed.
func (m *manager) evaluateSlowPath(s *state, req *admission.AdmissionRequest, deployment *storage.Deployment, start time.Time) (*admission.AdmissionResponse, error) {
	// Slow path: skip entirely if there are no policies that require image enrichment data.
	if len(s.slowPathDeployDetector.PolicySet().GetCompiledPolicies()) == 0 {
		log.Debugf("No policies violated, allowing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		observeAdmissionReview(reviewResultAllowed, time.Since(start))
		return pass(req.UID), nil
	}

	// Slow path: Fetch images and evaluate slow path policies. If there are no modified images, skip the scan.
	var fetchImgCtx context.Context
	if timeoutSecs := s.GetClusterConfig().GetAdmissionControllerConfig().GetTimeoutSeconds(); timeoutSecs > 1 && hasModifiedImages(s, deployment, req) {
		var cancel context.CancelFunc
		fetchImgCtx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
		defer cancel()
	}

	getAlertsFunc := func(dep *storage.Deployment, imgs []*storage.Image) ([]*storage.Alert, error) {
		return s.slowPathDeployDetector.Detect(context.Background(), detectionCtx, booleanpolicy.EnhancedDeployment{
			Deployment: dep,
			Images:     imgs,
		})
	}

	alerts, err := m.kickOffImgScansAndDetect(fetchImgCtx, s, getAlertsFunc, deployment)
	if err != nil {
		observeAdmissionReview(reviewResultError, time.Since(start))
		return nil, errors.Wrapf(err, "running %s detection", branding.GetProductNameShort())
	}

	if len(alerts) == 0 {
		log.Debugf("No policies violated, allowing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		observeAdmissionReview(reviewResultAllowed, time.Since(start))
		return pass(req.UID), nil
	}

	if !pointer.BoolDeref(req.DryRun, false) {
		go m.filterAndPutAttemptedAlertsOnChan(req.Operation, alerts...)
	}

	log.Debugf("Violated policies: %d, rejecting %s request on %s/%s [%s]", len(alerts), req.Operation, req.Namespace, req.Name, req.Kind)
	observeAdmissionReview(reviewResultDenied, time.Since(start))
	return fail(req.UID, message(alerts, !s.GetClusterConfig().GetAdmissionControllerConfig().GetDisableBypass(), 0)), nil
}

func toPlaceholderImages(deployment *storage.Deployment) []*storage.Image {
	images := make([]*storage.Image, len(deployment.GetContainers()))
	for i, c := range deployment.GetContainers() {
		images[i] = types.ToImage(c.GetImage())
	}
	return images
}
