package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScanSetting handles compliance operator scan setting.
type ScanSetting struct {
}

// NewScanSettingDispatcher creates and returns a new scan setting dispatcher.
func NewScanSettingDispatcher() *ScanSetting {
	return &ScanSetting{}
}

// ProcessEvent processes a scan setting event.
func (c *ScanSetting) ProcessEvent(obj, _ interface{}, _ central.ResourceAction) *component.ResourceEvent {
	var scanSetting v1alpha1.ScanSetting

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &scanSetting); err != nil {
		log.Errorf("error converting unstructured to compliance scan setting result: %v", err)
		return nil
	}

	// TODO: [ROX-18579] Add workflow to handle scan setting sensor events.
	return component.NewEvent()
}
