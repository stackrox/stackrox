package resources

import (
	"testing"

	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/registrymirror"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
			ms := &FakeMirrorStore{}
			d := newRegistryMirrorDispatcher(ms)

			d.ProcessEvent(icspA, nil, tc.action)
			assert.True(t, ms.upsertICSPInvoked)
			assert.False(t, ms.deleteICSPInvoked)

			d.ProcessEvent(idmsA, nil, tc.action)
			assert.True(t, ms.upsertIDMSInvoked)
			assert.False(t, ms.deleteIDMSInvoked)

			d.ProcessEvent(itmsA, nil, tc.action)
			assert.True(t, ms.upsertITMSInvoked)
			assert.False(t, ms.deleteITMSInvoked)
		})
	}

	t.Run("no panic when removing non existent resource", func(t *testing.T) {
		ms := &FakeMirrorStore{}
		d := newRegistryMirrorDispatcher(ms)
		action := central.ResourceAction_REMOVE_RESOURCE
		d.ProcessEvent(icspA, nil, action)
		assert.False(t, ms.upsertICSPInvoked)
		assert.True(t, ms.deleteICSPInvoked)

		d.ProcessEvent(idmsA, nil, action)
		assert.False(t, ms.upsertIDMSInvoked)
		assert.True(t, ms.deleteIDMSInvoked)

		d.ProcessEvent(itmsA, nil, action)
		assert.False(t, ms.upsertITMSInvoked)
		assert.True(t, ms.deleteITMSInvoked)
	})

	t.Run("unknown type panics for non-release builds", func(t *testing.T) {
		if !buildinfo.ReleaseBuild {
			ms := &FakeMirrorStore{}
			d := newRegistryMirrorDispatcher(ms)
			assert.Panics(t, func() { d.ProcessEvent(nil, nil, 0) })
		}
	})
}

type FakeMirrorStore struct {
	deleteICSPInvoked bool
	deleteIDMSInvoked bool
	deleteITMSInvoked bool

	upsertICSPInvoked bool
	upsertIDMSInvoked bool
	upsertITMSInvoked bool
}

var _ registrymirror.Store = (*FakeMirrorStore)(nil)

func (*FakeMirrorStore) Cleanup()                                           {}
func (*FakeMirrorStore) DeleteImageContentSourcePolicy(uid types.UID) error { return nil }
func (*FakeMirrorStore) DeleteImageDigestMirrorSet(uid types.UID) error     { return nil }
func (*FakeMirrorStore) DeleteImageTagMirrorSet(uid types.UID) error        { return nil }
func (*FakeMirrorStore) UpsertImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet) error {
	return nil
}
func (*FakeMirrorStore) UpsertImageTagMirrorSet(itms *configV1.ImageTagMirrorSet) error { return nil }
func (*FakeMirrorStore) UpsertImageContentSourcePolicy(icsp *operatorV1Alpha1.ImageContentSourcePolicy) error {
	return nil
}
func (*FakeMirrorStore) PullSources(srcImage string) ([]string, error) {
	return nil, nil
}
func (*FakeMirrorStore) UpdateConfig(icspRules []*operatorV1Alpha1.ImageContentSourcePolicy, idmsRules []*configV1.ImageDigestMirrorSet, itmsRules []*configV1.ImageTagMirrorSet) error {
	return nil
}
func (*FakeMirrorStore) GetAllMirrorSets() ([]*operatorV1Alpha1.ImageContentSourcePolicy, []*configV1.ImageDigestMirrorSet, []*configV1.ImageTagMirrorSet) {
	return nil, nil, nil
}
