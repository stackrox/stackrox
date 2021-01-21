package manager

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/stringutils"
	admission "k8s.io/api/admission/v1beta1"
)

func (m *manager) shouldBypassRuntimeDetection(s *state, req *admission.AdmissionRequest) bool {
	if s.runtimeDetector == nil {
		log.Debugf("Runtime policy matcher not found, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
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
		if kubernetes.SystemNamespaceSet.Contains(saNamespace) {
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

func (m *manager) evaluateRuntimeAdmissionRequest(s *state, req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	log.Debugf("Evaluating request %+v", req)
	if !features.K8sEventDetection.Enabled() {
		return pass(req.UID), nil
	}

	if m.shouldBypassRuntimeDetection(s, req) {
		return pass(req.UID), nil
	}

	log.Debugf("Not bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)

	event, err := kubernetes.AdmissionRequestToKubeEventObj(req)
	if err != nil {
		return nil, errors.Wrap(err, "translating admission request object from request")
	}

	log.Debugf("Evaluating policies on %s", kubernetes.EventAsString(event))

	// TODO: Map pods to deployment

	alerts, err := s.runtimeDetector.DetectForDeployment(&storage.Deployment{}, nil, nil, false, event)
	if err != nil {
		return nil, errors.Wrap(err, "running StackRox detection")
	}

	if len(alerts) == 0 {
		log.Debugf("No policies violated, allowing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return pass(req.UID), nil
	}

	atleastOneEnforced := false
	for _, alert := range alerts {
		if alert.GetEnforcement().GetAction() == storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT {
			atleastOneEnforced = true
			break
		}
	}

	if atleastOneEnforced {
		return fail(req.UID, message(alerts, false)), nil
	}
	// TODO: Mark enforced violations as attempted
	go m.putAlertsOnChan(alerts)

	return pass(req.UID), nil
}
