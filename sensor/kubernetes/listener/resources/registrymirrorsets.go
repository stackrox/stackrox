package resources

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/containers/image/v5/pkg/sysregistriesv2"
	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/openshift/runtime-utils/pkg/registries"
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
		r.debouncedUpdate()
		return nil
	}

	r.registryStore.UpsertImageContentSourcePolicy(icsp)
	log.Debugf("Upserted ImageContentSourcePolicy into registry store: %q (%v)", icsp.GetName(), icsp.GetUID())
	r.debouncedUpdate()
	return nil
}

func (r *registryMirrorDispatcher) handleImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet, action central.ResourceAction) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		r.registryStore.DeleteImageDigestMirrorSet(idms.GetUID())
		log.Debugf("Deleted ImageDigestMirrorSet from registry store: %q (%v)", idms.GetName(), idms.GetUID())
		r.debouncedUpdate()
		return nil
	}

	r.registryStore.UpsertImageDigestMirrorSet(idms)
	log.Debugf("Upserted ImageDigestMirrorSet into registry store: %q (%v)", idms.GetName(), idms.GetUID())
	r.debouncedUpdate()
	return nil
}

func (r *registryMirrorDispatcher) handleImageTagMirrorSet(itms *configV1.ImageTagMirrorSet, action central.ResourceAction) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		r.registryStore.DeleteImageTagMirrorSet(itms.GetUID())
		log.Debugf("Deleted ImageTagMirrorSet from registry store: %q (%v)", itms.GetName(), itms.GetUID())
		r.debouncedUpdate()
		return nil
	}

	r.registryStore.UpsertImageTagMirrorSet(itms)
	log.Debugf("Upserted ImageTagMirrorSet into registry store: %q (%v)", itms.GetName(), itms.GetUID())
	r.debouncedUpdate()
	return nil
}

// TOOD: update names
// TODO: abstract 'persister / updater'

// triggerDelayedUpdate
func (r *registryMirrorDispatcher) debouncedUpdate() {
	// TODO: delay
	err := r.doUpdate()
	if err != nil {
		log.Errorf("Updating consolidated registries: %v", err)
	} else {
		log.Debugf("Successfully wrote consolidated mirroring config")
	}
}

func (r *registryMirrorDispatcher) doUpdate() error {
	// update store
	// TODO: Make me a const
	path := "/var/cache/stackrox/mirrors/registries.conf"

	icspRules, idmsRules, itmsRules := r.registryStore.GetAllMirrorSets()

	config := new(sysregistriesv2.V2RegistriesConf)
	err := registries.EditRegistriesConfig(config, nil, nil, icspRules, idmsRules, itmsRules)
	if err != nil {
		return err
	}

	var newData bytes.Buffer
	encoder := toml.NewEncoder(&newData)
	if err := encoder.Encode(config); err != nil {
		return nil
	}

	// ensure dir exists
	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	// write the consolidate file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(newData.Bytes())
	if err != nil {
		return err
	}

	return nil
}
