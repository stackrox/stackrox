package complianceoperator

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/adhocore/gronx"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/sensor/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	return &v1alpha1.ScanSetting{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ScanSetting.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:        request.GetScanName(),
			Namespace:   namespace,
			Labels:      utils.GetSensorKubernetesLabels(),
			Annotations: utils.GetSensorKubernetesAnnotations(),
		},
		Roles: []string{masterRole, workerRole},
		ComplianceSuiteSettings: v1alpha1.ComplianceSuiteSettings{
			AutoApplyRemediations:  false,
			AutoUpdateRemediations: false,
			Schedule:               cron,
		},
		ComplianceScanSettings: v1alpha1.ComplianceScanSettings{
			StrictNodeScan:    pointers.Bool(false),
			ShowNotApplicable: false,
			Timeout:           env.ComplianceScanTimeout.Setting(),
			MaxRetryOnTimeout: env.ComplianceScanRetries.IntegerSetting(),
		},
	}
}

func convertCentralRequestToScanSettingBinding(namespace string, request *central.ApplyComplianceScanConfigRequest_BaseScanSettings, _ string) runtime.Object {
	profileRefs := make([]v1alpha1.NamedObjectReference, 0, len(request.GetProfiles()))
	for _, profile := range request.GetProfiles() {
		profileRefs = append(profileRefs, v1alpha1.NamedObjectReference{
			Name:     profile,
			Kind:     complianceoperator.Profile.Kind,
			APIGroup: complianceoperator.GetGroupVersion().String(),
		})
	}

	return &v1alpha1.ScanSettingBinding{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ScanSettingBinding.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:        request.GetScanName(),
			Namespace:   namespace,
			Labels:      utils.GetSensorKubernetesLabels(),
			Annotations: utils.GetSensorKubernetesAnnotations(),
		},
		Profiles: profileRefs,
		SettingsRef: &v1alpha1.NamedObjectReference{
			Name:     request.GetScanName(),
			Kind:     complianceoperator.ScanSetting.Kind,
			APIGroup: complianceoperator.GetGroupVersion().String(),
		},
	}
}

func updateScanSettingFromCentralRequest(scanSetting *v1alpha1.ScanSetting, request *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) *v1alpha1.ScanSetting {
	// TODO:  Update additional fields as ACS capability expands
	scanSetting.Roles = []string{masterRole, workerRole}
	scanSetting.ComplianceSuiteSettings = v1alpha1.ComplianceSuiteSettings{
		AutoApplyRemediations:  false,
		AutoUpdateRemediations: false,
		Schedule:               request.GetCron(),
	}
	scanSetting.ComplianceScanSettings = v1alpha1.ComplianceScanSettings{
		StrictNodeScan:    pointers.Bool(false),
		ShowNotApplicable: false,
		Timeout:           env.ComplianceScanTimeout.Setting(),
		MaxRetryOnTimeout: env.ComplianceScanRetries.IntegerSetting(),
	}

	return scanSetting
}

func updateScanSettingBindingFromCentralRequest(scanSettingBinding *v1alpha1.ScanSettingBinding, request *central.ApplyComplianceScanConfigRequest_BaseScanSettings) *v1alpha1.ScanSettingBinding {
	profileRefs := make([]v1alpha1.NamedObjectReference, 0, len(request.GetProfiles()))
	for _, profile := range request.GetProfiles() {
		profileRefs = append(profileRefs, v1alpha1.NamedObjectReference{
			Name:     profile,
			Kind:     complianceoperator.Profile.Kind,
			APIGroup: complianceoperator.GetGroupVersion().String(),
		})
	}

	// TODO:  Update additional fields as ACS capability expands
	scanSettingBinding.Profiles = profileRefs

	return scanSettingBinding
}

func validateApplyScheduledScanConfigRequest(req *central.ApplyComplianceScanConfigRequest_ScheduledScan) error {
	if req == nil {
		return errors.New("apply scan configuration request is empty")
	}
	var errList errorhelpers.ErrorList
	if req.GetScanSettings().GetScanName() == "" {
		errList.AddStrings("no name provided for the scan")
	}
	if len(req.GetScanSettings().GetProfiles()) == 0 {
		errList.AddStrings("compliance profiles not specified")
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
	if len(req.GetScanSettings().GetProfiles()) == 0 {
		errList.AddStrings("compliance profiles not specified")
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
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, errors.Wrap(err, "converting to unstructured object")
	}

	return &unstructured.Unstructured{
		Object: unstructuredObj,
	}, nil
}

func updateScanSettingFromUpdateRequest(obj *unstructured.Unstructured, req *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) (*unstructured.Unstructured, error) {
	var scanSetting v1alpha1.ScanSetting
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &scanSetting); err != nil {
		return nil, errors.Wrap(err, "Could not convert unstructured to scan setting")
	}

	return runtimeObjToUnstructured(updateScanSettingFromCentralRequest(&scanSetting, req))
}

func updateScanSettingBindingFromUpdateRequest(obj *unstructured.Unstructured, req *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) (*unstructured.Unstructured, error) {
	var scanSettingBinding v1alpha1.ScanSettingBinding
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &scanSettingBinding); err != nil {
		return nil, errors.Wrap(err, "Could not convert unstructured to scan setting")
	}

	return runtimeObjToUnstructured(updateScanSettingBindingFromCentralRequest(&scanSettingBinding, req.GetScanSettings()))
}
