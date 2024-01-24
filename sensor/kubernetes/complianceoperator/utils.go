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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

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
func convertCentralRequestToScanSetting(namespace string, request *central.ApplyComplianceScanConfigRequest_ScheduledScan) *v1alpha1.ScanSetting {
	// TODO: Add ACS labels.
	return &v1alpha1.ScanSetting{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ScanSetting.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      request.GetScanSettings().GetScanName(),
			Namespace: namespace,
		},
		Roles: []string{masterRole, workerRole},
		ComplianceSuiteSettings: v1alpha1.ComplianceSuiteSettings{
			AutoApplyRemediations:  false,
			AutoUpdateRemediations: false,
			Schedule:               request.GetCron(),
		},
		ComplianceScanSettings: v1alpha1.ComplianceScanSettings{
			StrictNodeScan:    pointers.Bool(false),
			ShowNotApplicable: false,
			Timeout:           env.ComplianceScanTimeout.Setting(),
			MaxRetryOnTimeout: env.ComplianceScanRetries.IntegerSetting(),
		},
	}
}

func convertCentralRequestToScanSettingBinding(namespace string, request *central.ApplyComplianceScanConfigRequest_BaseScanSettings) *v1alpha1.ScanSettingBinding {
	profileRefs := make([]v1alpha1.NamedObjectReference, 0, len(request.GetProfiles()))
	for _, profile := range request.GetProfiles() {
		profileRefs = append(profileRefs, v1alpha1.NamedObjectReference{
			Name:     profile,
			Kind:     complianceoperator.Profile.Kind,
			APIGroup: complianceoperator.GetGroupVersion().String(),
		})
	}

	// TODO: Add ACS labels.
	return &v1alpha1.ScanSettingBinding{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ScanSettingBinding.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      request.GetScanName(),
			Namespace: namespace,
		},
		Profiles: profileRefs,
		SettingsRef: &v1alpha1.NamedObjectReference{
			Name:     request.GetScanName(),
			Kind:     complianceoperator.ScanSetting.Kind,
			APIGroup: complianceoperator.GetGroupVersion().String(),
		},
	}
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
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: unstructuredObj,
	}, nil
}
