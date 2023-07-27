package resources

import (
	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type registryMirrorDispatcher struct {
}

func newRegistryMirrorDispatcher() *registryMirrorDispatcher {
	return &registryMirrorDispatcher{}
}

// ProcessEvent processes registry mirroring related resource events and returns the sensor events to emit in response.
func (r *registryMirrorDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	switch v := obj.(type) {
	case *operatorV1Alpha1.ImageContentSourcePolicy:
		return r.handleImageContentSourcePolicy(v, action)
	case *configV1.ImageDigestMirrorSet:
		return r.handleImageDigestMirrorSet(v, action)
	case *configV1.ImageTagMirrorSet:
		return r.handleImageTagMirrorSet(v, action)
	}

	return nil
}

func (r *registryMirrorDispatcher) handleImageContentSourcePolicy(icsp *operatorV1Alpha1.ImageContentSourcePolicy, action central.ResourceAction) *component.ResourceEvent {
	// TODO(ROX-18251 & ROX-18248): will be implemented as part of the referenced stories.
	log.Debugf("Received registry mirror ImageContentSourcePolicy event [%v]: %#v", action.String(), icsp)
	return nil
}

func (r *registryMirrorDispatcher) handleImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet, action central.ResourceAction) *component.ResourceEvent {
	// TODO(ROX-18251 & ROX-18248): will be implemented as part of the referenced stories.
	log.Debugf("Received registry mirror ImageDigestMirrorSet event [%v]: %#v", action.String(), idms)
	return nil
}

func (r *registryMirrorDispatcher) handleImageTagMirrorSet(itms *configV1.ImageTagMirrorSet, action central.ResourceAction) *component.ResourceEvent {
	// TODO(ROX-18251 & ROX-18248): will be implemented as part of the referenced stories.
	log.Debugf("Received registry mirror ImageTagMirrorSet event [%v]: %#v", action.String(), itms)
	return nil
}
