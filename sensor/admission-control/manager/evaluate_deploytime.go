package manager

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/stringutils"
	admission "k8s.io/api/admission/v1"
	"k8s.io/utils/pointer"
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

	// We don't enforce on subresources.
	if req.SubResource != "" {
		log.Debugf("Request is for a subresource, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
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

// hasNonNoScanAlerts checks if the given alert slice contains any alerts that are NOT
// due to the absence (or presence) of image scans.
func hasNonNoScanAlerts(alerts []*storage.Alert) bool {
	for _, a := range alerts {
		if !policyfields.ContainsScanRequiredFields(a.GetPolicy()) {
			return true
		}
	}
	return false
}

// filterOutNoScanAlerts removes all alerts from the given slice that are due to the absence
// of image scans. The given slice is modified and should not be used afterwards.
func filterOutNoScanAlerts(alerts []*storage.Alert) []*storage.Alert {
	filteredAlerts := alerts[:0]
	for _, a := range alerts {
		if policyfields.ContainsScanRequiredFields(a.GetPolicy()) {
			continue
		}
		filteredAlerts = append(filteredAlerts, a)
	}
	return filteredAlerts
}

func (m *manager) evaluateAdmissionRequest(s *state, req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	log.Debugf("Evaluating request %+v", req)

	if m.shouldBypass(s, req) {
		return pass(req.UID), nil
	}

	log.Debugf("Not bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)

	k8sObj, err := unmarshalK8sObject(req.Kind, req.Object.Raw)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal object from request")
	}

	deployment, err := resources.NewDeploymentFromStaticResource(k8sObj, req.Kind.Kind, s.clusterID(), s.GetClusterConfig().GetRegistryOverride())
	if err != nil {
		return nil, errors.Wrap(err, "could not convert Kubernetes object into StackRox deployment")
	}

	if deployment == nil {
		log.Debugf("Non-top-level object, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return pass(req.UID), nil // we only enforce on top-level objects
	}

	log.Debugf("Evaluating policies on %+v", deployment)

	// Check if the deployment has a bypass annotation
	if !s.GetClusterConfig().GetAdmissionControllerConfig().GetDisableBypass() {
		if !enforcers.ShouldEnforce(deployment.GetAnnotations()) {
			log.Warnf("deployment %s/%s of type %v was deployed without being checked due to matching bypass annotation %q",
				deployment.GetNamespace(), deployment.GetName(), req.Kind, enforcers.EnforcementBypassAnnotationKey)
			return pass(req.UID), nil
		}
	}

	var fetchImgCtx context.Context
	if timeoutSecs := s.GetClusterConfig().GetAdmissionControllerConfig().GetTimeoutSeconds(); timeoutSecs > 1 && hasModifiedImages(s, deployment, req) {
		var cancel context.CancelFunc
		fetchImgCtx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
		defer cancel()
	}

	getAlertsFunc := func(dep *storage.Deployment, imgs []*storage.Image) ([]*storage.Alert, error) {
		return s.deploytimeDetector.Detect(detectionCtx, booleanpolicy.EnhancedDeployment{
			Deployment: dep,
			Images:     imgs,
		})
	}

	alerts, err := m.kickOffImgScansAndDetect(fetchImgCtx, s, getAlertsFunc, deployment)
	if err != nil {
		return nil, errors.Wrap(err, "running StackRox detection")
	}

	if len(alerts) == 0 {
		log.Debugf("No policies violated, allowing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return pass(req.UID), nil
	}

	if !pointer.BoolDeref(req.DryRun, false) {
		go m.filterAndPutAttemptedAlertsOnChan(req.Operation, alerts...)
	}

	log.Debugf("Violated policies: %d, rejecting %s request on %s/%s [%s]", len(alerts), req.Operation, req.Namespace, req.Name, req.Kind)
	return fail(req.UID, message(alerts, !s.GetClusterConfig().GetAdmissionControllerConfig().GetDisableBypass())), nil
}
