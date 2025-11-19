package defaults

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"k8s.io/utils/ptr"
)

const (
	FeatureDefaultKeyAdmissionControllerEnforcement = "feature-defaults.platform.stackrox.io/admissionControllerEnforcement"
)

var SecuredClusterAdmissionControllerDefaultingFlow = SecuredClusterDefaultingFlow{
	Name:           "secured-cluster-admission-controller",
	DefaultingFunc: securedClusterAdmissionControllerDefaulting,
}

func admissionControllerDefaultingGreenField(logger logr.Logger, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	return admissionControllerDefaultingGreenFieldEnforce(logger, annotations, spec, defaults)
}

func admissionControllerDefaultingGreenFieldEnforce(_ logr.Logger, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	admissionControl := spec.AdmissionControl
	if admissionControl == nil {
		admissionControl = &platform.AdmissionControlComponentSpec{}
	}
	if admissionControl.Enforcement != nil {
		return nil
	}

	enforcement := platform.PolicyEnforcementEnabled
	defaults.AdmissionControl.Enforcement = ptr.To(enforcement)
	annotations[FeatureDefaultKeyAdmissionControllerEnforcement] = string(enforcement)
	return nil
}

func admissionControllerDefaultingBrownFieldEnforce(logger logr.Logger, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	var functionErr error
	admissionControl := spec.AdmissionControl
	if admissionControl == nil {
		admissionControl = &platform.AdmissionControlComponentSpec{}
	}

	if enforceAnnotation := annotations[FeatureDefaultKeyAdmissionControllerEnforcement]; enforceAnnotation != "" {
		if !(enforceAnnotation == "Enabled" || enforceAnnotation == "Disabled") {
			logger.Info("Invalid CR defaulting annotation {%q: %v}", FeatureDefaultKeyAdmissionControllerEnforcement, enforceAnnotation)
			functionErr = errors.Errorf("unexpected value %q of CR annotation %s", enforceAnnotation, FeatureDefaultKeyAdmissionControllerEnforcement)
		} else {
			defaults.AdmissionControl.Enforcement = ptr.To(platform.PolicyEnforcement(enforceAnnotation))
		}
	}

	if defaults.AdmissionControl.Enforcement == nil {
		// No previous annotation, implement defaulting flow.
		// Note: we don't have fields "enforceOnCreates" and "enforceOnUpdated" in the CRD.
		// These fields for the Helm chart were historically kept in sync with the "listenOnCreates" and the "listenOnUpdates" fields of the CRD.
		listenOnCreates := false
		listenOnUpdates := false

		if listenOnCreatesPtr := admissionControl.ListenOnCreates; listenOnCreatesPtr != nil {
			listenOnCreates = *listenOnCreatesPtr
		}
		if listenOnUpdatesPtr := admissionControl.ListenOnUpdates; listenOnUpdatesPtr != nil {
			listenOnUpdates = *listenOnUpdatesPtr
		}

		enforcement := platform.PolicyEnforcementDisabled
		if listenOnCreates || listenOnUpdates {
			enforcement = platform.PolicyEnforcementEnabled
		}
		defaults.AdmissionControl.Enforcement = ptr.To(enforcement)
	}

	if enforcement := defaults.AdmissionControl.Enforcement; enforcement != nil {
		enforcementString := string(*enforcement)
		if annotations[FeatureDefaultKeyAdmissionControllerEnforcement] != enforcementString {
			annotations[FeatureDefaultKeyAdmissionControllerEnforcement] = enforcementString
		}
	}

	return functionErr
}

func admissionControllerDefaultingBrownField(logger logr.Logger, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	return admissionControllerDefaultingBrownFieldEnforce(logger, annotations, spec, defaults)
}

func securedClusterAdmissionControllerDefaulting(logger logr.Logger, status *platform.SecuredClusterStatus, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	if securedClusterStatusUninitialized(status) {
		// Green field.
		logger.Info("Assuming new installation due to empty status.")
		if err := admissionControllerDefaultingGreenField(logger, annotations, spec, defaults); err != nil {
			return err
		}
	} else {
		// Brown field.
		logger.Info("Assuming upgrade due to populated status.")
		if err := admissionControllerDefaultingBrownField(logger, annotations, spec, defaults); err != nil {
			return err
		}
	}

	return nil
}
