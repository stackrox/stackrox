package resources

import (
	"os"
	"path/filepath"
	"testing"

	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProcessEvent(t *testing.T) {
	icspA := &operatorV1Alpha1.ImageContentSourcePolicy{ObjectMeta: v1.ObjectMeta{Name: "icspA", UID: "UIDicspA"}}
	itmsA := &configV1.ImageDigestMirrorSet{ObjectMeta: v1.ObjectMeta{Name: "itmsA", UID: "UIDitmsA"}}
	idmsA := &configV1.ImageTagMirrorSet{ObjectMeta: v1.ObjectMeta{Name: "idmsA", UID: "UIDidmsA"}}

	// Ensure all actions (except delete) result in upserts
	tt := []struct {
		action central.ResourceAction
	}{
		{central.ResourceAction_UNSET_ACTION_RESOURCE},
		{central.ResourceAction_SYNC_RESOURCE},
		{central.ResourceAction_CREATE_RESOURCE},
		{central.ResourceAction_UPDATE_RESOURCE},
	}
	for _, tc := range tt {
		t.Run(tc.action.String(), func(t *testing.T) {
			rs := registry.NewRegistryStore(nil)
			d := newRegistryMirrorDispatcher(rs)
			d.ProcessEvent(icspA, nil, tc.action)
			d.ProcessEvent(idmsA, nil, tc.action)
			d.ProcessEvent(itmsA, nil, tc.action)
			icspRules, idmsRules, itmsRules := rs.GetAllMirrorSets()
			assert.Len(t, icspRules, 1)
			assert.Len(t, idmsRules, 1)
			assert.Len(t, itmsRules, 1)
		})
	}

	t.Run("no panic when removing non existant resource", func(t *testing.T) {
		rs := registry.NewRegistryStore(nil)
		d := newRegistryMirrorDispatcher(rs)
		action := central.ResourceAction_REMOVE_RESOURCE
		d.ProcessEvent(icspA, nil, action)
		d.ProcessEvent(idmsA, nil, action)
		d.ProcessEvent(itmsA, nil, action)
	})

	t.Run("upsert followed by delete removes appropriate resource", func(t *testing.T) {
		rs := registry.NewRegistryStore(nil)
		d := newRegistryMirrorDispatcher(rs)
		action := central.ResourceAction_CREATE_RESOURCE
		d.ProcessEvent(icspA, nil, action)
		d.ProcessEvent(idmsA, nil, action)
		d.ProcessEvent(itmsA, nil, action)

		action = central.ResourceAction_REMOVE_RESOURCE
		d.ProcessEvent(icspA, nil, action)
		icspRules, idmsRules, itmsRules := rs.GetAllMirrorSets()
		assert.Len(t, icspRules, 0)
		assert.Len(t, idmsRules, 1)
		assert.Len(t, itmsRules, 1)

		d.ProcessEvent(idmsA, nil, action)
		icspRules, idmsRules, itmsRules = rs.GetAllMirrorSets()
		assert.Len(t, icspRules, 0)
		assert.Len(t, idmsRules, 1)
		assert.Len(t, itmsRules, 0)

		d.ProcessEvent(itmsA, nil, action)
		icspRules, idmsRules, itmsRules = rs.GetAllMirrorSets()
		assert.Len(t, icspRules, 0)
		assert.Len(t, idmsRules, 0)
		assert.Len(t, itmsRules, 0)
	})

	t.Run("unknown type panics for non-release builds", func(t *testing.T) {
		if !buildinfo.ReleaseBuild {
			rs := registry.NewRegistryStore(nil)
			d := newRegistryMirrorDispatcher(rs)
			assert.Panics(t, func() { d.ProcessEvent(nil, nil, 0) })
		}
	})
}

func TestDoWrite(t *testing.T) {
	path := "/Users/dcaravel/dev/stackrox/stackrox-mirror-filesys/dignore/containers/registries.conf"

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Error(err)
	}
}
