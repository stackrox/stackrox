package complianceoperator

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/errorhelpers"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func convertCentralRequestToScanSettingBinding(namespace string, request *central.ApplyComplianceScanConfigRequest_OneTimeScan) *v1alpha1.ScanSettingBinding {
	profileRefs := make([]v1alpha1.NamedObjectReference, 0, len(request.GetProfiles()))
	for _, profile := range request.GetProfiles() {
		profileRefs = append(profileRefs, v1alpha1.NamedObjectReference{
			Name:     profile,
			Kind:     complianceoperator.ScanSettingGVK.Kind,
			APIGroup: complianceoperator.GetGroupVersion().String(),
		})
	}

	// TODO: Add ACS labels.
	return &v1alpha1.ScanSettingBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      request.GetScanSettings().GetScanName(),
			Namespace: namespace,
		},
		Profiles: profileRefs,
		SettingsRef: &v1alpha1.NamedObjectReference{
			Name:     defaultScanSettingName,
			Kind:     complianceoperator.ScanSettingGVK.Kind,
			APIGroup: complianceoperator.GetGroupVersion().String(),
		},
	}
}

func validateApplyOneTimeScanConfigRequest(req *central.ApplyComplianceScanConfigRequest_OneTimeScan) error {
	if req == nil {
		return errors.New("apply scan configuration request is empty")
	}
	var errList errorhelpers.ErrorList
	if req.GetScanSettings().GetScanName() == "" {
		errList.AddStrings("no name provided for the scan")
	}
	if len(req.GetProfiles()) == 0 {
		errList.AddStrings("compliance profiles not specified")
	}
	return errList.ToError()
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
