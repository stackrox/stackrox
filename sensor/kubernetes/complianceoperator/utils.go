package complianceoperator

import (
	"github.com/adhocore/gronx"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/sensor/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	stackroxLabels = map[string]string{
		"app.kubernetes.io/name": "stackrox",
	}
)

// convertFunction signature of the convert functions
type convertFunction func(string, *central.ApplyComplianceScanConfigRequest_BaseScanSettings, string) runtime.Object

// namedObjectReference mirrors v1alpha1.NamedObjectReference without importing that package.
type namedObjectReference struct {
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	APIGroup string `json:"apiGroup,omitempty"`
}

// profileKindToString maps an OperatorKind proto enum to the compliance operator Kind string.
func profileKindToString(kind central.ComplianceOperatorProfileV2_OperatorKind) string {
	switch kind {
	case central.ComplianceOperatorProfileV2_PROFILE:
		return complianceoperator.Profile.Kind
	case central.ComplianceOperatorProfileV2_TAILORED_PROFILE:
		return complianceoperator.TailoredProfile.Kind
	case central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}

func validateScanSettingBindingProfiles(scanSettings *central.ApplyComplianceScanConfigRequest_BaseScanSettings) error {
	if scanSettings == nil {
		return nil
	}
	if len(scanSettings.GetProfiles()) == 0 && len(scanSettings.GetProfileRefs()) == 0 {
		return errors.New("compliance profiles not specified")
	}
	for _, ref := range scanSettings.GetProfileRefs() {
		k := ref.GetKind()
		if k != central.ComplianceOperatorProfileV2_PROFILE && k != central.ComplianceOperatorProfileV2_TAILORED_PROFILE {
			err := errors.Errorf("profile ref %q has unsupported operator kind %v", ref.GetName(), k)
			return err
		}
	}
	return nil
}

func buildScanSettingBindingProfileRefs(namespace string, request *central.ApplyComplianceScanConfigRequest_BaseScanSettings) []namedObjectReference {
	// Profiles may be provided via request.ProfileRefs, where each reference contains a profile name and its kind, or
	// via the legacy request.Profiles, which only contains a slice of profile names. When both are present, ProfileRefs
	// take precedence.
	profileRefs := make([]namedObjectReference, 0, len(request.GetProfileRefs()))
	for _, ref := range request.GetProfileRefs() {
		profileRefs = append(profileRefs, namedObjectReference{
			Name:     ref.GetName(),
			Kind:     profileKindToString(ref.GetKind()),
			APIGroup: complianceoperator.GetGroupVersion().String(),
		})
	}
	if len(profileRefs) > 0 {
		log.Debugf("Using %d profile_refs from Central for namespace %q", len(profileRefs), namespace)
		return profileRefs
	}

	// Legacy: old Central without profile_refs — default to Profile kind.
	profileRefs = make([]namedObjectReference, 0, len(request.GetProfiles()))
	for _, profile := range request.GetProfiles() {
		profileRefs = append(profileRefs, namedObjectReference{
			Name:     profile,
			Kind:     complianceoperator.Profile.Kind,
			APIGroup: complianceoperator.GetGroupVersion().String(),
		})
	}
	log.Debugf("Using legacy profiles (%d) for namespace %q", len(request.GetProfiles()), namespace)
	return profileRefs
}

// updateFunction signature of the update functions
type updateFunction func(*unstructured.Unstructured, *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) (*unstructured.Unstructured, error)

type scanNameGetter interface {
	GetScanName() string
}

func validateScanName(req scanNameGetter) error {
	if req == nil {
		return errors.New("apply scan configuration request is empty")
	}
	if req.GetScanName() == "" {
		return errors.New("no name provided for the scan")
	}
	return nil
}

func convertCentralRequestToScanSetting(namespace string, request *central.ApplyComplianceScanConfigRequest_BaseScanSettings, cron string) runtime.Object {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": complianceoperator.GetGroupVersion().String(),
			"kind":       complianceoperator.ScanSetting.Kind,
			"metadata": map[string]interface{}{
				"name":        request.GetScanName(),
				"namespace":   namespace,
				"labels":      toStringInterfaceMap(utils.GetSensorKubernetesLabels()),
				"annotations": toStringInterfaceMap(utils.GetSensorKubernetesAnnotations()),
			},
			"roles":                  []interface{}{masterRole, workerRole},
			"autoApplyRemediations":  false,
			"autoUpdateRemediations": false,
			"schedule":               cron,
			"strictNodeScan":         false,
			"showNotApplicable":      false,
			"timeout":                env.ComplianceScanTimeout.Setting(),
			"maxRetryOnTimeout":      int64(env.ComplianceScanRetries.IntegerSetting()),
		},
	}
	return obj
}

func convertCentralRequestToScanSettingBinding(namespace string, request *central.ApplyComplianceScanConfigRequest_BaseScanSettings, _ string) runtime.Object {
	profileRefs := buildScanSettingBindingProfileRefs(namespace, request)

	profiles := make([]interface{}, 0, len(profileRefs))
	for _, ref := range profileRefs {
		profiles = append(profiles, map[string]interface{}{
			"name":     ref.Name,
			"kind":     ref.Kind,
			"apiGroup": ref.APIGroup,
		})
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": complianceoperator.GetGroupVersion().String(),
			"kind":       complianceoperator.ScanSettingBinding.Kind,
			"metadata": map[string]interface{}{
				"name":        request.GetScanName(),
				"namespace":   namespace,
				"labels":      toStringInterfaceMap(utils.GetSensorKubernetesLabels()),
				"annotations": toStringInterfaceMap(utils.GetSensorKubernetesAnnotations()),
			},
			"profiles": profiles,
			"settingsRef": map[string]interface{}{
				"name":     request.GetScanName(),
				"kind":     complianceoperator.ScanSetting.Kind,
				"apiGroup": complianceoperator.GetGroupVersion().String(),
			},
		},
	}
	return obj
}

func updateScanSettingFromUpdateRequest(obj *unstructured.Unstructured, req *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) (*unstructured.Unstructured, error) {
	// Update fields in-place on the unstructured object.
	obj.Object["roles"] = []interface{}{masterRole, workerRole}
	obj.Object["autoApplyRemediations"] = false
	obj.Object["autoUpdateRemediations"] = false
	obj.Object["schedule"] = req.GetCron()
	obj.Object["strictNodeScan"] = false
	obj.Object["showNotApplicable"] = false
	obj.Object["timeout"] = env.ComplianceScanTimeout.Setting()
	obj.Object["maxRetryOnTimeout"] = int64(env.ComplianceScanRetries.IntegerSetting())

	return obj, nil
}

func updateScanSettingBindingFromUpdateRequest(obj *unstructured.Unstructured, req *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) (*unstructured.Unstructured, error) {
	profileRefs := buildScanSettingBindingProfileRefs(obj.GetNamespace(), req.GetScanSettings())

	profiles := make([]interface{}, 0, len(profileRefs))
	for _, ref := range profileRefs {
		profiles = append(profiles, map[string]interface{}{
			"name":     ref.Name,
			"kind":     ref.Kind,
			"apiGroup": ref.APIGroup,
		})
	}
	obj.Object["profiles"] = profiles

	return obj, nil
}

func validateApplyScheduledScanConfigRequest(req *central.ApplyComplianceScanConfigRequest_ScheduledScan) error {
	if req == nil {
		return errors.New("apply scan configuration request is empty")
	}
	var errList errorhelpers.ErrorList
	if req.GetScanSettings().GetScanName() == "" {
		errList.AddStrings("no name provided for the scan")
	}
	if err := validateScanSettingBindingProfiles(req.GetScanSettings()); err != nil {
		errList.AddError(err)
	}
	if req.GetCron() != "" {
		cron := gronx.New()
		if !cron.IsValid(req.GetCron()) {
			errList.AddStrings("schedule is not valid")
		}
	}
	return errList.ToError()
}

func validateUpdateScheduledScanConfigRequest(req *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) error {
	if req == nil {
		return errors.New("update scan configuration request is empty")
	}
	var errList errorhelpers.ErrorList
	if req.GetScanSettings().GetScanName() == "" {
		errList.AddStrings("no name provided for the scan")
	}
	if err := validateScanSettingBindingProfiles(req.GetScanSettings()); err != nil {
		errList.AddError(err)
	}
	if req.GetCron() != "" {
		cron := gronx.New()
		if !cron.IsValid(req.GetCron()) {
			errList.AddStrings("schedule is not valid")
		}
	}
	return errList.ToError()
}

func validateApplySuspendScheduledScanRequest(req *central.ApplyComplianceScanConfigRequest_SuspendScheduledScan) error {
	return validateScanName(req)
}

func validateApplyResumeScheduledScanRequest(req *central.ApplyComplianceScanConfigRequest_ResumeScheduledScan) error {
	return validateScanName(req)
}

func validateApplyRerunScheduledScanRequest(req *central.ApplyComplianceScanConfigRequest_RerunScheduledScan) error {
	if req == nil {
		return errors.New("apply scan configuration request is empty")
	}
	if req.GetScanName() == "" {
		return errors.New("no name provided for the scan")
	}
	return nil
}

func runtimeObjToUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	// If the object is already unstructured, return it directly.
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u, nil
	}
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, errors.Wrap(err, "converting to unstructured object")
	}

	return &unstructured.Unstructured{
		Object: unstructuredObj,
	}, nil
}

// toStringInterfaceMap converts map[string]string to map[string]interface{} for unstructured objects.
func toStringInterfaceMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
