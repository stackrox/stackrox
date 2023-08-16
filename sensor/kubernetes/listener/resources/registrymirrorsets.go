package resources

import (
	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/registrymirror"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type registryMirrorDispatcher struct {
	mirrorStore registrymirror.Store
}

func newRegistryMirrorDispatcher(mirrorStore registrymirror.Store) *registryMirrorDispatcher {
	return &registryMirrorDispatcher{
		mirrorStore: mirrorStore,
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
		if err := r.mirrorStore.DeleteImageContentSourcePolicy(icsp.GetUID()); err != nil {
			log.Errorf("Deleting ImageContentSourcePolicy from mirror store %q (%v): %v", icsp.GetName(), icsp.GetUID(), err)
		} else {
			log.Debugf("Deleted ImageContentSourcePolicy from mirror store: %q (%v)", icsp.GetName(), icsp.GetUID())
		}
		return nil
	}

	if err := r.mirrorStore.UpsertImageContentSourcePolicy(icsp); err != nil {
		log.Errorf("Upserting ImageContentSourcePolicy into mirror store %q (%v): %v", icsp.GetName(), icsp.GetUID(), err)
	} else {
		log.Debugf("Upserted ImageContentSourcePolicy into mirror store: %q (%v)", icsp.GetName(), icsp.GetUID())
	}
	return nil
}

func (r *registryMirrorDispatcher) handleImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet, action central.ResourceAction) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		if err := r.mirrorStore.DeleteImageDigestMirrorSet(idms.GetUID()); err != nil {
			log.Errorf("Deleting ImageDigestMirrorSet from mirror store %q (%v): %v", idms.GetName(), idms.GetUID(), err)
		} else {
			log.Debugf("Deleted ImageDigestMirrorSet from mirror store: %q (%v)", idms.GetName(), idms.GetUID())
		}
		return nil
	}

	if err := r.mirrorStore.UpsertImageDigestMirrorSet(idms); err != nil {
		log.Errorf("Upserting ImageDigestMirrorSet into mirror store %q (%v): %v", idms.GetName(), idms.GetUID(), err)
	} else {
		log.Debugf("Upserted ImageDigestMirrorSet into mirror store: %q (%v)", idms.GetName(), idms.GetUID())
	}
	return nil
}

func (r *registryMirrorDispatcher) handleImageTagMirrorSet(itms *configV1.ImageTagMirrorSet, action central.ResourceAction) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		if err := r.mirrorStore.DeleteImageTagMirrorSet(itms.GetUID()); err != nil {
			log.Errorf("Deleting ImageTagMirrorSet from mirror store %q (%v): %v", itms.GetName(), itms.GetUID(), err)
		} else {
			log.Debugf("Deleted ImageTagMirrorSet from mirror store: %q (%v)", itms.GetName(), itms.GetUID())
		}
		return nil
	}

	if err := r.mirrorStore.UpsertImageTagMirrorSet(itms); err != nil {
		log.Errorf("Upserting ImageTagMirrorSet into mirror store %q (%v): %v", itms.GetName(), itms.GetUID(), err)
	} else {
		log.Debugf("Upserted ImageTagMirrorSet into mirror store: %q (%v)", itms.GetName(), itms.GetUID())
	}
	return nil
}
