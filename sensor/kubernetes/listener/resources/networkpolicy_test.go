package resources

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestGetSelector(t *testing.T) {
	if !features.NetworkPolicySystemPolicy.Enabled() {
		t.Skipf("Skipping test since the %s variable is not set", features.NetworkPolicySystemPolicy.EnvVar())
	}

	mockCtrl := gomock.NewController(t)
	nps := newNetworkPoliciesStore()
	ds := newDeploymentStore()
	det := mocks.NewMockDetector(mockCtrl)

	dispatcher := newNetworkPolicyDispatcher(nps, ds, det)

	cases := []struct {
		name             string
		netpol           *storage.NetworkPolicy
		oldNetpol        *storage.NetworkPolicy
		action           central.ResourceAction
		expectedSelector []map[string]string
		expectedEmpty    bool
	}{
		{
			name: "New NetworkPolicy",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app":  "sensor",
							"role": "backend",
						},
					},
				},
			},
			oldNetpol: nil,
			action:    central.ResourceAction_CREATE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			expectedEmpty: false,
		},
		{
			name: "New NetworkPolicy, no selector",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			oldNetpol:        nil,
			action:           central.ResourceAction_CREATE_RESOURCE,
			expectedSelector: []map[string]string{},
			expectedEmpty:    true,
		},
		{
			name: "Update NetworkPolicy",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app":  "sensor",
							"role": "backend",
						},
					},
				},
			},
			oldNetpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app":  "sensor-2",
							"role": "backend",
						},
					},
				},
			},
			action: central.ResourceAction_UPDATE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
				{
					"app":  "sensor-2",
					"role": "backend",
				},
			},
			expectedEmpty: false,
		},
		{
			name: "Update NetworkPolicy, no selector",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			oldNetpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			action:           central.ResourceAction_UPDATE_RESOURCE,
			expectedSelector: []map[string]string{},
			expectedEmpty:    true,
		},
		{
			name: "Update NetworkPolicy, new selector",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app":  "sensor",
							"role": "backend",
						},
					},
				},
			},
			oldNetpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			action: central.ResourceAction_UPDATE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			expectedEmpty: true,
		},
		{
			name: "Update NetworkPolicy, delete selector",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			oldNetpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app":  "sensor",
							"role": "backend",
						},
					},
				},
			},
			action: central.ResourceAction_UPDATE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			expectedEmpty: true,
		},
		{
			name: "Delete NetworkPolicy",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app":  "sensor",
							"role": "backend",
						},
					},
				},
			},
			oldNetpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app":  "sensor",
							"role": "backend",
						},
					},
				},
			},
			action: central.ResourceAction_REMOVE_RESOURCE,
			expectedSelector: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			expectedEmpty: false,
		},
		{
			name: "Delete NetworkPolicy, no selector",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			oldNetpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			action:           central.ResourceAction_REMOVE_RESOURCE,
			expectedSelector: []map[string]string{},
			expectedEmpty:    true,
		},
	}
	for _, c := range cases {
		sel, isEmpty := dispatcher.getSelector(c.netpol, c.oldNetpol, c.action)
		assert.Equal(t, isEmpty, c.expectedEmpty)
		for _, s := range c.expectedSelector {
			assert.True(t, sel.Matches(labels.Set(s)))
		}
	}
}

func TestUpdateDeploymentsFromStore(t *testing.T) {
	if !features.NetworkPolicySystemPolicy.Enabled() {
		t.Skipf("Skipping test since the %s variable is not set", features.NetworkPolicySystemPolicy.EnvVar())
	}

	mockCtrl := gomock.NewController(t)
	nps := newNetworkPoliciesStore()
	ds := newDeploymentStore()
	det := mocks.NewMockDetector(mockCtrl)

	dispatcher := newNetworkPolicyDispatcher(nps, ds, det)

	deployments := []*deploymentWrap{
		{
			Deployment: &storage.Deployment{
				Name:      "deploy-1",
				Id:        "1",
				Namespace: "default",
				PodLabels: map[string]string{
					"app":  "sensor",
					"role": "backend",
				},
			},
		},
		{
			Deployment: &storage.Deployment{
				Name:      "deploy-2",
				Id:        "2",
				Namespace: "default",
				PodLabels: map[string]string{},
			},
		},
		{
			Deployment: &storage.Deployment{
				Name:      "deploy-3",
				Id:        "3",
				Namespace: "secure",
				PodLabels: map[string]string{
					"app":  "sensor",
					"role": "backend",
				},
			},
		},
		{
			Deployment: &storage.Deployment{
				Name:      "deploy-4",
				Id:        "4",
				Namespace: "default",
				PodLabels: map[string]string{
					"app": "sensor-2",
				},
			},
		},
	}

	for _, d := range deployments {
		ds.addOrUpdateDeployment(d)
	}

	cases := []struct {
		name                string
		netpol              *storage.NetworkPolicy
		sel                 []map[string]string
		isEmpty             bool
		expectedDeployments []*deploymentWrap
	}{
		{
			name: "New NetworkPolicy",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			sel: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
			},
			isEmpty: false,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
			},
		},
		{
			name: "Empty selector",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			sel:     []map[string]string{},
			isEmpty: true,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "2",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
		{
			name: "Selector with no deployments",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			sel: []map[string]string{
				{
					"app": "central",
				},
			},
			isEmpty:             false,
			expectedDeployments: []*deploymentWrap{},
		},
		{
			name: "Namespace with no deployments",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "random_namespace",
			},
			sel: []map[string]string{
				{
					"app": "sensor",
				},
			},
			isEmpty:             false,
			expectedDeployments: []*deploymentWrap{},
		},
		{
			name: "Namespace with no deployments, no selector",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "random_namespace",
			},
			sel:                 []map[string]string{},
			isEmpty:             true,
			expectedDeployments: []*deploymentWrap{},
		},
		{
			name: "Disjunction selector",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			sel: []map[string]string{
				{
					"app":  "sensor",
					"role": "backend",
				},
				{
					"app": "sensor-2",
				},
			},
			isEmpty: false,
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
		{
			name: "Disjunction selector, with empty member",
			netpol: &storage.NetworkPolicy{
				Id:        "1",
				Namespace: "default",
			},
			sel: []map[string]string{
				{},
				{
					"app": "sensor-2",
				},
			},
			isEmpty: true, // If one of the members of the selector is empty the selector is considered empty
			expectedDeployments: []*deploymentWrap{
				{
					Deployment: &storage.Deployment{
						Id:        "1",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "2",
						Namespace: "default",
					},
				},
				{
					Deployment: &storage.Deployment{
						Id:        "4",
						Namespace: "default",
					},
				},
			},
		},
	}
	for _, c := range cases {
		deps := map[string]*deploymentWrap{}
		processDeploymentMock := det.EXPECT().ProcessDeployment(gomock.Any(), central.ResourceAction_UPDATE_RESOURCE).DoAndReturn(func(d *storage.Deployment, _ central.ResourceAction) {
			deps[d.GetId()] = &deploymentWrap{
				Deployment: d,
			}
		})
		processDeploymentMock.Times(len(c.expectedDeployments))
		var sel selector
		for _, s := range c.sel {
			if sel != nil {
				sel = or(sel, SelectorFromMap(s))
			} else {
				sel = SelectorFromMap(s)
			}
		}
		dispatcher.updateDeploymentsFromStore(c.netpol, sel, c.isEmpty)
		for _, d := range c.expectedDeployments {
			_, ok := deps[d.GetId()]
			assert.True(t, ok)
		}
	}
}
