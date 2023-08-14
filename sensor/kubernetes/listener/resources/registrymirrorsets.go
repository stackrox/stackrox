package resources

import (
	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type registryMirrorDispatcher struct {
	registryStore *registry.Store
}

func newRegistryMirrorDispatcher(registryStore *registry.Store) *registryMirrorDispatcher {
	return &registryMirrorDispatcher{
		registryStore: registryStore,
	}
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
	default:
		utils.Should(errors.Errorf("Unsupported type: %T", v))
	}

	return nil
}

func (r *registryMirrorDispatcher) handleImageContentSourcePolicy(icsp *operatorV1Alpha1.ImageContentSourcePolicy, action central.ResourceAction) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		r.registryStore.DeleteImageContentSourcePolicy(icsp.GetUID())
		log.Debugf("Deleted ImageContentSourcePolicy from registry store: %q (%v)", icsp.GetName(), icsp.GetUID())
		return nil
	}

	r.registryStore.UpsertImageContentSourcePolicy(icsp)
	log.Debugf("Upserted ImageContentSourcePolicy into registry store: %q (%v)", icsp.GetName(), icsp.GetUID())
	return nil
}

func (r *registryMirrorDispatcher) handleImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet, action central.ResourceAction) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		r.registryStore.DeleteImageDigestMirrorSet(idms.GetUID())
		log.Debugf("Deleted ImageDigestMirrorSet from registry store: %q (%v)", idms.GetName(), idms.GetUID())
		return nil
	}

	r.registryStore.UpsertImageDigestMirrorSet(idms)
	log.Debugf("Upserted ImageDigestMirrorSet into registry store: %q (%v)", idms.GetName(), idms.GetUID())
	return nil
}

func (r *registryMirrorDispatcher) handleImageTagMirrorSet(itms *configV1.ImageTagMirrorSet, action central.ResourceAction) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		r.registryStore.DeleteImageTagMirrorSet(itms.GetUID())
		log.Debugf("Deleted ImageTagMirrorSet from registry store: %q (%v)", itms.GetName(), itms.GetUID())
		return nil
	}

	r.registryStore.UpsertImageTagMirrorSet(itms)
	log.Debugf("Upserted ImageTagMirrorSet into registry store: %q (%v)", itms.GetName(), itms.GetUID())
	return nil
}
