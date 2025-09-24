package defaults

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"k8s.io/utils/ptr"
)

const (
	FeatureDefaultKeyAdmissionControllerEnforce = "feature-defaults.platform.stackrox.io/admissionControllerEnforce"
)

var SecuredClusterAdmissionControllerDefaultingFlow = SecuredClusterDefaultingFlow{
	Name:           "secured-cluster-admission-controller",
	DefaultingFunc: securedClusterAdmissionControllerDefaulting,
}

var (
	tableBoolMarshalling = map[bool]string{
		true:  "true",
		false: "false",
	}
	tableBoolUnmarshalling = map[string]bool{
		"true":  true,
		"false": false,
	}
)

func admissionControllerDefaultingGreenField(logger logr.Logger, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	return admissionControllerDefaultingGreenFieldEnforce(logger, annotations, spec, defaults)
}

func admissionControllerDefaultingGreenFieldEnforce(_ logr.Logger, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	admissionControl := spec.AdmissionControl
	if admissionControl == nil {
		admissionControl = &platform.AdmissionControlComponentSpec{}
	}
	if admissionControl.Enforce != nil {
		return nil
	}

	enforceBool := true
	defaults.AdmissionControl.Enforce = ptr.To(enforceBool)
	annotations[FeatureDefaultKeyAdmissionControllerEnforce] = tableBoolMarshalling[enforceBool]
	return nil
}

func admissionControllerDefaultingBrownFieldEnforce(logger logr.Logger, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	var functionErr error
	admissionControl := spec.AdmissionControl
	if admissionControl == nil {
		admissionControl = &platform.AdmissionControlComponentSpec{}
	}

	if enforceAnnotation := annotations[FeatureDefaultKeyAdmissionControllerEnforce]; enforceAnnotation != "" {
		enforceBool, ok := tableBoolUnmarshalling[enforceAnnotation]
		if !ok {
			logger.Info("Failed to unmarshal CR defaulting annotation {%q: %v} as boolean", FeatureDefaultKeyAdmissionControllerEnforce, enforceAnnotation)
			functionErr = errors.Errorf("unexpected value %q of CR annotation %s", enforceAnnotation, FeatureDefaultKeyAdmissionControllerEnforce)
		} else {
			defaults.AdmissionControl.Enforce = ptr.To(enforceBool)
		}
	}

	if defaults.AdmissionControl.Enforce == nil {
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
		defaults.AdmissionControl.Enforce = ptr.To(listenOnCreates || listenOnUpdates)
	}

	if defaults.AdmissionControl.Enforce != nil {
		enforceString := tableBoolMarshalling[*defaults.AdmissionControl.Enforce]
		if annotations[FeatureDefaultKeyAdmissionControllerEnforce] != enforceString {
			annotations[FeatureDefaultKeyAdmissionControllerEnforce] = enforceString
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
