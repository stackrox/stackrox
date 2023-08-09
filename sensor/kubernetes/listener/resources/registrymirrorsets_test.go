package resources

import (
	"testing"

	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/registrymirror/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProcessEvent(t *testing.T) {
	icspA := &operatorV1Alpha1.ImageContentSourcePolicy{ObjectMeta: v1.ObjectMeta{Name: "icspA", UID: "UIDicspA"}}
	idmsA := &configV1.ImageDigestMirrorSet{ObjectMeta: v1.ObjectMeta{Name: "idmsA", UID: "UIDidmsA"}}
	itmsA := &configV1.ImageTagMirrorSet{ObjectMeta: v1.ObjectMeta{Name: "itmsA", UID: "UIDitmsA"}}

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
			ms := mocks.NewMockStore(gomock.NewController(t))
			d := newRegistryMirrorDispatcher(ms)

			ms.EXPECT().UpsertImageContentSourcePolicy(icspA)
			d.ProcessEvent(icspA, nil, tc.action)

			ms.EXPECT().UpsertImageDigestMirrorSet(idmsA)
			d.ProcessEvent(idmsA, nil, tc.action)

			ms.EXPECT().UpsertImageTagMirrorSet(itmsA)
			d.ProcessEvent(itmsA, nil, tc.action)
		})
	}

	t.Run("no panic when removing non existent resource", func(t *testing.T) {
		ms := mocks.NewMockStore(gomock.NewController(t))
		d := newRegistryMirrorDispatcher(ms)

		action := central.ResourceAction_REMOVE_RESOURCE

		ms.EXPECT().DeleteImageContentSourcePolicy(icspA.UID)
		d.ProcessEvent(icspA, nil, action)

		ms.EXPECT().DeleteImageDigestMirrorSet(idmsA.UID)
		d.ProcessEvent(idmsA, nil, action)

		ms.EXPECT().DeleteImageTagMirrorSet(itmsA.UID)
		d.ProcessEvent(itmsA, nil, action)
	})

	t.Run("unknown type panics for non-release builds", func(t *testing.T) {
		if !buildinfo.ReleaseBuild {
			ms := mocks.NewMockStore(gomock.NewController(t))
			d := newRegistryMirrorDispatcher(ms)
			assert.Panics(t, func() { d.ProcessEvent(nil, nil, 0) })
		}
	})
}
