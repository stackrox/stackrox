package rbac

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestStore_DispatcherEvents(t *testing.T) {
	// Namespace: n1
	// Role: r1
	// Bindings:
	//  - b1 -> r1
	//  - b2 -> r1
	// Cluster role: r2
	// Cluster bindings:
	//  - b3 -> r2
	//  - b4 -> r2
	roles := []*v1.Role{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("r1"),
				Name:      "r1",
				Namespace: "n1",
			},
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"get"},
			}, {
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"list"},
			}},
		},
	}
	clusterRoles := []*v1.ClusterRole{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:  types.UID("r2"),
				Name: "r2",
			},
		},
	}
	bindings := []*v1.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("b1"),
				Name:      "b1",
				Namespace: "n1",
			},
			RoleRef: v1.RoleRef{
				Name:     "r1",
				Kind:     "Role",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("b2"),
				Name:      "b2",
				Namespace: "n1",
			},
			RoleRef: v1.RoleRef{
				Name:     "r1",
				Kind:     "Role",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("b5"),
				Name:      "b5",
				Namespace: "n1",
			},
			RoleRef: v1.RoleRef{
				Name:     "r2",
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}
	clusterBindings := []*v1.ClusterRoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:  types.UID("b3"),
				Name: "b3",
			},
			RoleRef: v1.RoleRef{
				Name:     "r2",
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:  types.UID("b4"),
				Name: "b4",
			},
			RoleRef: v1.RoleRef{
				Name:     "r2",
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}

	tested := NewStore().(*storeImpl)
	fakeClient := fake.NewSimpleClientset()
	dispatcher := NewDispatcher(tested, fakeClient)

	eventsInOrder := []struct {
		k8sEvent          any
		action            central.ResourceAction
		unorderedMessages []*central.SensorEvent
		createK8sResource func() error
	}{
		{
			k8sEvent: bindings[0],
			action:   central.ResourceAction_CREATE_RESOURCE,
			createK8sResource: func() error {
				_, err := fakeClient.RbacV1().RoleBindings(bindings[0].Namespace).Create(context.TODO(), bindings[0], metav1.CreateOptions{})
				return err
			},
			unorderedMessages: []*central.SensorEvent{
				central.SensorEvent_builder{
					Id:     "b1",
					Action: central.ResourceAction_CREATE_RESOURCE,
					Binding: storage.K8SRoleBinding_builder{
						Id:        "b1",
						Name:      "b1",
						Namespace: "n1",
						// No role ID since the role does not yet exist.
						CreatedAt: protoconv.ConvertTimeToTimestamp(bindings[0].GetCreationTimestamp().Time),
						Subjects:  []*storage.Subject{},
					}.Build(),
				}.Build()},
		},
		{
			k8sEvent: roles[0],
			action:   central.ResourceAction_CREATE_RESOURCE,
			createK8sResource: func() error {
				_, err := fakeClient.RbacV1().Roles(roles[0].Namespace).Create(context.TODO(), roles[0], metav1.CreateOptions{})
				return err
			},
			unorderedMessages: []*central.SensorEvent{
				central.SensorEvent_builder{
					Id:     "r1",
					Action: central.ResourceAction_CREATE_RESOURCE,
					Role: storage.K8SRole_builder{
						Id:        "r1",
						Name:      "r1",
						Namespace: "n1",
						CreatedAt: protoconv.ConvertTimeToTimestamp(roles[0].GetCreationTimestamp().Time),
						Rules: []*storage.PolicyRule{storage.PolicyRule_builder{
							ApiGroups: []string{""},
							Resources: []string{""},
							Verbs:     []string{"get"},
						}.Build(), storage.PolicyRule_builder{
							ApiGroups: []string{""},
							Resources: []string{""},
							Verbs:     []string{"list"},
						}.Build()},
					}.Build(),
				}.Build(),
				central.SensorEvent_builder{
					Id:     "b1",
					Action: central.ResourceAction_UPDATE_RESOURCE,
					Binding: storage.K8SRoleBinding_builder{
						Id:        "b1",
						Name:      "b1",
						Namespace: "n1",
						CreatedAt: protoconv.ConvertTimeToTimestamp(bindings[0].GetCreationTimestamp().Time),
						RoleId:    "r1",
						Subjects:  []*storage.Subject{},
					}.Build(),
				}.Build()},
		},
		{
			k8sEvent: bindings[1],
			action:   central.ResourceAction_CREATE_RESOURCE,
			createK8sResource: func() error {
				_, err := fakeClient.RbacV1().RoleBindings(bindings[1].Namespace).Create(context.TODO(), bindings[1], metav1.CreateOptions{})
				return err
			},
			unorderedMessages: []*central.SensorEvent{
				central.SensorEvent_builder{
					Id:     "b2",
					Action: central.ResourceAction_CREATE_RESOURCE,
					Binding: storage.K8SRoleBinding_builder{
						Id:        "b2",
						Name:      "b2",
						Namespace: "n1",
						RoleId:    "r1", // Note that the role ID is now filled in.
						CreatedAt: protoconv.ConvertTimeToTimestamp(bindings[1].GetCreationTimestamp().Time),
						Subjects:  []*storage.Subject{},
					}.Build(),
				}.Build(),
			},
		},
		{
			k8sEvent: bindings[2],
			action:   central.ResourceAction_CREATE_RESOURCE,
			createK8sResource: func() error {
				_, err := fakeClient.RbacV1().RoleBindings(bindings[2].Namespace).Create(context.TODO(), bindings[2], metav1.CreateOptions{})
				return err
			},
			unorderedMessages: []*central.SensorEvent{
				central.SensorEvent_builder{
					Id:     "b5",
					Action: central.ResourceAction_CREATE_RESOURCE,
					Binding: storage.K8SRoleBinding_builder{
						Id:          "b5",
						Name:        "b5",
						Namespace:   "n1",
						RoleId:      "",
						ClusterRole: true,
						CreatedAt:   protoconv.ConvertTimeToTimestamp(bindings[2].GetCreationTimestamp().Time),
						Subjects:    []*storage.Subject{},
					}.Build(),
				}.Build(),
			},
		},
		{
			k8sEvent: clusterBindings[0],
			action:   central.ResourceAction_CREATE_RESOURCE,
			createK8sResource: func() error {
				_, err := fakeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterBindings[0], metav1.CreateOptions{})
				return err
			},
			unorderedMessages: []*central.SensorEvent{central.SensorEvent_builder{
				Id:     "b3",
				Action: central.ResourceAction_CREATE_RESOURCE,
				Binding: storage.K8SRoleBinding_builder{
					Id:   "b3",
					Name: "b3",
					// No role ID since the role does not yet exist.
					ClusterRole: true,
					CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterBindings[0].GetCreationTimestamp().Time),
					Subjects:    []*storage.Subject{},
				}.Build(),
			}.Build()},
		},
		{
			k8sEvent: clusterRoles[0],
			action:   central.ResourceAction_CREATE_RESOURCE,
			createK8sResource: func() error {
				_, err := fakeClient.RbacV1().ClusterRoles().Create(context.TODO(), clusterRoles[0], metav1.CreateOptions{})
				return err
			},
			unorderedMessages: []*central.SensorEvent{
				central.SensorEvent_builder{
					Id:     "r2",
					Action: central.ResourceAction_CREATE_RESOURCE,
					Role: storage.K8SRole_builder{
						Id:          "r2",
						Name:        "r2",
						ClusterRole: true,
						CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterRoles[0].GetCreationTimestamp().Time),
						Rules:       []*storage.PolicyRule{},
					}.Build(),
				}.Build(),
				central.SensorEvent_builder{
					Id:     "b5",
					Action: central.ResourceAction_UPDATE_RESOURCE,
					Binding: storage.K8SRoleBinding_builder{
						Id:          "b5",
						Name:        "b5",
						Namespace:   "n1",
						RoleId:      "r2",
						ClusterRole: true,
						CreatedAt:   protoconv.ConvertTimeToTimestamp(bindings[2].GetCreationTimestamp().Time),
						Subjects:    []*storage.Subject{},
					}.Build(),
				}.Build(),
				central.SensorEvent_builder{
					Id:     "b3",
					Action: central.ResourceAction_UPDATE_RESOURCE,
					Binding: storage.K8SRoleBinding_builder{
						Id:          "b3",
						Name:        "b3",
						ClusterRole: true,
						RoleId:      "r2",
						CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterBindings[0].GetCreationTimestamp().Time),
						Subjects:    []*storage.Subject{},
					}.Build(),
				}.Build()},
		},
		{
			k8sEvent: bindings[2],
			action:   central.ResourceAction_UPDATE_RESOURCE,
			createK8sResource: func() error {
				_, err := fakeClient.RbacV1().RoleBindings(bindings[2].Namespace).Update(context.TODO(), bindings[2], metav1.UpdateOptions{})
				return err
			},
			unorderedMessages: []*central.SensorEvent{central.SensorEvent_builder{
				Id:     "b5",
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Binding: storage.K8SRoleBinding_builder{
					Id:          "b5",
					Name:        "b5",
					Namespace:   "n1",
					RoleId:      "r2",
					ClusterRole: true,
					CreatedAt:   protoconv.ConvertTimeToTimestamp(bindings[2].GetCreationTimestamp().Time),
					Subjects:    []*storage.Subject{},
				}.Build(),
			}.Build()},
		},
		{
			k8sEvent: clusterBindings[0],
			action:   central.ResourceAction_UPDATE_RESOURCE,
			createK8sResource: func() error {
				_, err := fakeClient.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterBindings[0], metav1.UpdateOptions{})
				return err
			},
			unorderedMessages: []*central.SensorEvent{central.SensorEvent_builder{
				Id:     "b3",
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Binding: storage.K8SRoleBinding_builder{
					Id:          "b3",
					Name:        "b3",
					RoleId:      "r2", // Note that the role ID is now filled in.
					ClusterRole: true,
					CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterBindings[0].GetCreationTimestamp().Time),
					Subjects:    []*storage.Subject{},
				}.Build(),
			}.Build()},
		},
		{
			k8sEvent: clusterRoles[0],
			action:   central.ResourceAction_REMOVE_RESOURCE,
			createK8sResource: func() error {
				return fakeClient.RbacV1().ClusterRoles().Delete(context.TODO(), clusterRoles[0].Name, metav1.DeleteOptions{})
			},
			unorderedMessages: []*central.SensorEvent{central.SensorEvent_builder{
				Id:     "r2",
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Role: storage.K8SRole_builder{
					Id:          "r2",
					Name:        "r2",
					ClusterRole: true,
					CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterRoles[0].GetCreationTimestamp().Time),
					Rules:       []*storage.PolicyRule{},
				}.Build(),
			}.Build(),
				central.SensorEvent_builder{
					Id:     "b5",
					Action: central.ResourceAction_UPDATE_RESOURCE,
					Binding: storage.K8SRoleBinding_builder{
						Id:        "b5",
						Name:      "b5",
						Namespace: "n1",
						// Note that the role ID is now absent.
						ClusterRole: true,
						CreatedAt:   protoconv.ConvertTimeToTimestamp(bindings[2].GetCreationTimestamp().Time),
						Subjects:    []*storage.Subject{},
					}.Build(),
				}.Build(),
				central.SensorEvent_builder{
					Id:     "b3",
					Action: central.ResourceAction_UPDATE_RESOURCE,
					Binding: storage.K8SRoleBinding_builder{
						Id:   "b3",
						Name: "b3",
						// Note that the role ID is now absent.
						ClusterRole: true,
						CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterBindings[0].GetCreationTimestamp().Time),
						Subjects:    []*storage.Subject{},
					}.Build(),
				}.Build(),
			},
		},
	}

	for _, event := range eventsInOrder {
		require.NoError(t, event.createK8sResource())
		actual := dispatcher.ProcessEvent(event.k8sEvent, nil, event.action)
		protoassert.ElementsMatch(t, event.unorderedMessages, actual.ForwardMessages)
	}
}

func TestStore_DeploymentRelationship(t *testing.T) {
	roles := []*v1.Role{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("r1"),
				Name:      "r1",
				Namespace: "n1",
			},
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"get"},
			}},
		},
	}
	clusterRoles := []*v1.ClusterRole{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:  types.UID("r2"),
				Name: "r2",
			},
		},
	}
	bindings := []*v1.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("b1"),
				Name:      "b1",
				Namespace: "n1",
			},
			RoleRef: v1.RoleRef{
				Name:     "r1",
				Kind:     "Role",
				APIGroup: "rbac.authorization.k8s.io",
			},
			Subjects: []v1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "sa1",
				Namespace: "n1",
			}},
		},
	}
	clusterBindings := []*v1.ClusterRoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:  types.UID("b3"),
				Name: "b3",
			},
			RoleRef: v1.RoleRef{
				Name:     "r2",
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
			},
			Subjects: []v1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "sa2",
				Namespace: "n1",
			}},
		},
	}

	testCases := map[string]struct {
		orderedUpdates         []any
		serviceAccountsUpdates []string
	}{
		"Update service account based on single binding": {
			orderedUpdates:         []any{bindings[0]},
			serviceAccountsUpdates: []string{"sa1"},
		},
		"No service account update if only a role is received": {
			orderedUpdates:         []any{roles[0]},
			serviceAccountsUpdates: nil,
		},
		"Update service account both on binding and role update": {
			orderedUpdates:         []any{bindings[0], roles[0]},
			serviceAccountsUpdates: []string{"sa1", "sa1"},
		},
		"Update service account based on single cluster binding": {
			orderedUpdates:         []any{clusterBindings[0]},
			serviceAccountsUpdates: []string{"sa2"},
		},
		"No service account update if only a custer role is received": {
			orderedUpdates:         []any{clusterRoles[0]},
			serviceAccountsUpdates: nil,
		},
		"Update service account both on cluster binding and cluster role update": {
			orderedUpdates:         []any{clusterBindings[0], clusterRoles[0]},
			serviceAccountsUpdates: []string{"sa2", "sa2"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			tested := NewStore().(*storeImpl)
			fakeClient := fake.NewSimpleClientset()
			dispatcher := NewDispatcher(tested, fakeClient)
			var ref []resolver.DeploymentResolution
			for _, update := range testCase.orderedUpdates {
				event := dispatcher.ProcessEvent(update, nil, central.ResourceAction_CREATE_RESOURCE)
				if len(event.DeploymentReferences) == 1 {
					ref = append(ref, event.DeploymentReferences[0].Reference)
				}
			}

			var orderedServiceAccounts []string
			mockCtrl := gomock.NewController(t)
			deploymentStore := mocks.NewMockDeploymentStore(mockCtrl)

			deploymentStore.EXPECT().FindDeploymentIDsWithServiceAccount(gomock.Any(), gomock.Any()).
				Times(len(testCase.serviceAccountsUpdates)).
				Do(func(_, serviceAccount string) {
					orderedServiceAccounts = append(orderedServiceAccounts, serviceAccount)
				})

			for _, r := range ref {
				assert.NotNil(t, r)
				r(deploymentStore)
			}

			assert.Equal(t, testCase.serviceAccountsUpdates, orderedServiceAccounts)
		})

	}
}

type storeObjectCounts struct {
	roles      int
	bindings   int
	namespaces int
}

func (c storeObjectCounts) String() string {
	return fmt.Sprintf("Roles %v Bindings %v Namespaces %v", c.roles, c.bindings, c.namespaces)
}

// Taken from 4 customer debug dump metrics on 2021-09-09.
// The number of namespaces is unknown and entirely made up.
func getCustomerStoreObjectCounts() []storeObjectCounts {
	return []storeObjectCounts{
		{roles: 4_168, bindings: 5_281, namespaces: 50},
		{roles: 1_720, bindings: 10_306, namespaces: 100},
		{roles: 873, bindings: 66_258, namespaces: 500},
		{roles: 1_788, bindings: 351_582, namespaces: 1000},
	}
}

// Generates a store with the provided count of elements.
func generateStore(counts storeObjectCounts) Store {
	store := NewStore()

	for i := 0; i < counts.roles; i++ {
		store.UpsertRole(&v1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("some-role-%d", i),
				Namespace: fmt.Sprintf("some-namespace-%d", i%counts.namespaces),
				UID:       types.UID(uuid.NewV4().String()),
			},
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			}},
		})
	}
	for i := 0; i < counts.bindings; i++ {
		store.UpsertBinding(&v1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("some-binding-%d", i),
				Namespace: fmt.Sprintf("some-namespace-%d", i%counts.namespaces),
				UID:       types.UID(uuid.NewV4().String()),
			},
			RoleRef: v1.RoleRef{
				Name: fmt.Sprintf("some-role-%d", i%counts.roles),
			},
			Subjects: []v1.Subject{{
				Name:      "default-subject",
				Kind:      v1.ServiceAccountKind,
				Namespace: fmt.Sprintf("some-namespace-%d", i%counts.namespaces),
			}},
		})
	}

	return store
}

func BenchmarkRBACStoreUpsertTime(b *testing.B) {
	for n := 0; n < b.N; n++ {
		generateStore(storeObjectCounts{roles: 1000, bindings: 10_000, namespaces: 10})
	}
}

func runRBACBenchmarkGetPermissionLevelForDeployment(b *testing.B, store Store, keepCache bool) {
	for n := 0; n < b.N; n++ {
		deployment := &storage.Deployment{}
		deployment.SetServiceAccount("default-subject")
		deployment.SetNamespace("namespace0")
		store.GetPermissionLevelForDeployment(deployment)
		if !keepCache {
			// Important! We really want to call b.StopTimer() here and b.StartTimer() below the
			// UpsertRole call, but when we do this the Benchmarker hangs (see
			// https://stackoverflow.com/a/37624250 for more information). This means the UpsertRole
			// call will be included in the benchmark time.
			// Create a new role to trigger cache invalidation.
			store.UpsertRole(&v1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("roletoinvalidatecache%s", uuid.NewV4().String()),
					Namespace: "namespaceforcacheinvalidation",
					UID:       types.UID(uuid.NewV4().String()),
				},
				Rules: []v1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get"},
				}},
			})
		}
	}
}

func BenchmarkRBACStoreAssignPermissionLevelToDeployment(b *testing.B) {
	for _, keepCache := range []bool{true, false} {
		for _, warmUpCache := range []bool{true, false} {
			for _, counts := range getCustomerStoreObjectCounts() {
				store := generateStore(counts)
				if warmUpCache {
					deployment := &storage.Deployment{}
					deployment.SetServiceAccount("default-subject")
					deployment.SetNamespace("namespace0")
					_ = store.GetPermissionLevelForDeployment(deployment)
				}
				b.Run(fmt.Sprintf("KeepCache %t WarmUpCache %t %+v", keepCache, warmUpCache, counts), func(b *testing.B) {
					// The bucket evaluator is not built yet, we will build it initially
					// and keep using it.
					runRBACBenchmarkGetPermissionLevelForDeployment(b, store, keepCache)
				})
			}
		}
	}
}

func BenchmarkRBACUpsertExistingBinding(b *testing.B) {
	b.StopTimer()
	store := NewStore()
	binding := &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "role",
			Namespace: "namespace",
			UID:       types.UID(uuid.NewV4().String()),
		},
		RoleRef: v1.RoleRef{
			Name: "role",
		},
	}
	store.UpsertBinding(binding)
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		store.UpsertBinding(binding)
	}
}

func TestStoreGetPermissionLevelForDeployment(t *testing.T) {
	// This test creates roles and bindings and then updates them to match following state:
	// Roles:
	//  1. role-admin (all verbs on all resources)
	//  2. role-default (get)
	//  3. role-elevated (get, list) in 2 rules
	//  4. role-elevated-2 (get, list) in a single rule
	// Bindings:
	//  1. admin-subject      -> role-admin
	//  2. default-subject    -> role-default
	//  3. elevated-subject   -> role-elevated
	//  4. elevated-subject-2 -> role-elevated-2
	// Cluster Roles:
	//  1. cluster-admin (all verbs on all resources)
	//  2. cluster-elevated (get on all resources)
	//  3. cluster-elevated-2 (deletecollection)
	//  4. cluster-elevated-3 (deletecollection on pod duplicated)
	//  5. cluster-none (invalid verb on all resources in all API groups)
	//  6. cluster-elevated-admin (all verbs on all resources with additional rule)
	// Cluster Bindings:
	//  3. cluster-admin-subject    -> cluster-admin
	//  4. cluster-elevated-subject -> cluster-elevated
	//  5. cluster-elevated-admin   -> cluster-admin-2]
	//  6. cluster-elevated-2       -> cluster-elevated-subject-3
	//  7. cluster-elevated-3       -> cluster-elevated-subject-4
	//  8. cluster-none             -> cluster-none-subject
	roles := []*v1.Role{
		{
			ObjectMeta: meta("role-admin"),
		},
		{
			ObjectMeta: meta("role-default"),
		},
		{
			ObjectMeta: meta("role-admin"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}},
		},
		{
			ObjectMeta: meta("role-default"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"get"},
			}},
		},
		{
			ObjectMeta: meta("role-elevated"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"get"},
			}, {
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"list"},
			}},
		},
		{
			ObjectMeta: meta("role-elevated-2"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"get", "list"},
			}},
		},
	}
	bindings := []*v1.RoleBinding{
		{
			ObjectMeta: meta("b1"),
			RoleRef:    role("role-admin"),
		},
		{
			ObjectMeta: meta("b2"),
			RoleRef:    role("role-default"),
		},
		{
			ObjectMeta: meta("b1"),
			RoleRef:    role("role-admin"),
			Subjects: []v1.Subject{
				{
					Name:      "admin-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-namespace-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
		{
			ObjectMeta: meta("b2"),
			RoleRef:    role("role-default"),
			Subjects: []v1.Subject{{
				Name:      "default-subject",
				Kind:      v1.ServiceAccountKind,
				Namespace: "n1",
			}},
		},
		{
			ObjectMeta: meta("b3"),
			RoleRef:    role("role-elevated"),
			Subjects: []v1.Subject{{
				Name:      "elevated-subject",
				Kind:      v1.ServiceAccountKind,
				Namespace: "n1",
			}},
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    role("role-elevated-2"),
			Subjects: []v1.Subject{{
				Name:      "elevated-subject-2",
				Kind:      v1.ServiceAccountKind,
				Namespace: "n1",
			}},
		},
	}
	clusterRoles := []*v1.ClusterRole{
		{
			ObjectMeta: meta("cluster-admin"),
		},
		{
			ObjectMeta: meta("cluster-elevated"),
		},
		{
			ObjectMeta: meta("cluster-admin"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}},
		},
		{
			ObjectMeta: meta("cluster-elevated-2"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"deletecollection"},
			}},
		},
		{
			ObjectMeta: meta("cluster-elevated-3"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"pod"},
				Verbs:     []string{"deletecollection"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"pod"},
				Verbs:     []string{"deletecollection"},
			}},
		},
		{
			ObjectMeta: meta("cluster-none"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"invalidverb"},
			}},
		},
		{
			ObjectMeta: meta("cluster-elevated-admin"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}},
		},
		{
			ObjectMeta: meta("cluster-elevated"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get"},
			}},
		},
		{
			ObjectMeta: meta("cluster-elevated"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get"},
			}},
		},
	}
	clusterBindings := []*v1.ClusterRoleBinding{
		{
			ObjectMeta: meta("b3"),
			RoleRef:    clusterRole("cluster-admin"),
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    clusterRole("cluster-elevated"),
		},
		{
			ObjectMeta: meta("b3"),
			RoleRef:    clusterRole("cluster-admin"),
			Subjects: []v1.Subject{{
				Name: "cluster-admin-subject",
				Kind: v1.ServiceAccountKind,
			}},
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    clusterRole("cluster-elevated"),
			Subjects: []v1.Subject{
				{
					Name:      "cluster-elevated-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-elevated-subject-2",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-namespace-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
		{
			ObjectMeta: meta("b5"),
			RoleRef:    clusterRole("cluster-elevated-admin"),
			Subjects: []v1.Subject{
				{
					Name:      "cluster-admin-2",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
		{
			ObjectMeta: meta("b6"),
			RoleRef:    clusterRole("cluster-elevated-2"),
			Subjects: []v1.Subject{{
				Name: "cluster-elevated-subject-3",
				Kind: v1.ServiceAccountKind,
			}},
		},
		{
			ObjectMeta: meta("b7"),
			RoleRef:    clusterRole("cluster-elevated-3"),
			Subjects: []v1.Subject{{
				Name: "cluster-elevated-subject-4",
				Kind: v1.ServiceAccountKind,
			}},
		},
		{
			ObjectMeta: meta("b8"),
			RoleRef:    clusterRole("cluster-none"),
			Subjects: []v1.Subject{{
				Name: "cluster-none-subject",
				Kind: v1.ServiceAccountKind,
			}},
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    clusterRole("cluster-elevated"),
			Subjects: []v1.Subject{
				{
					Name:      "cluster-elevated-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-elevated-subject-2",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-namespace-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
	}

	testCases := []struct {
		deployment *storage.Deployment
		expected   storage.PermissionLevel
	}{
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-elevated-subject", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-elevated-subject-2", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-elevated-subject-3"}.Build()},
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-elevated-subject-4"}.Build()},
		{expected: storage.PermissionLevel_CLUSTER_ADMIN, deployment: storage.Deployment_builder{ServiceAccount: "cluster-admin-2", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-namespace-subject", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-elevated-subject"}.Build()},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-admin-subject", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_CLUSTER_ADMIN, deployment: storage.Deployment_builder{ServiceAccount: "cluster-admin-subject"}.Build()},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-none-subject"}.Build()},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment_builder{ServiceAccount: "cluster-none-subject", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_ELEVATED_IN_NAMESPACE, deployment: storage.Deployment_builder{ServiceAccount: "admin-subject", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_DEFAULT, deployment: storage.Deployment_builder{ServiceAccount: "default-subject", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_ELEVATED_IN_NAMESPACE, deployment: storage.Deployment_builder{ServiceAccount: "elevated-subject", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_ELEVATED_IN_NAMESPACE, deployment: storage.Deployment_builder{ServiceAccount: "elevated-subject-2", Namespace: "n1"}.Build()},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment_builder{ServiceAccount: "elevated-subject-2", Namespace: "n2"}.Build()},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment_builder{ServiceAccount: "default-subject"}.Build()},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment_builder{ServiceAccount: "admin-subject"}.Build()},
	}
	store := setupStore(roles, clusterRoles, bindings, clusterBindings)
	storeWithNoRoles := setupStore(roles, clusterRoles, bindings, clusterBindings)
	for _, r := range roles {
		storeWithNoRoles.RemoveRole(r)
	}
	for _, r := range clusterRoles {
		storeWithNoRoles.RemoveClusterRole(r)
	}
	storeWithNoBindings := setupStore(roles, clusterRoles, bindings, clusterBindings)
	for _, b := range bindings {
		storeWithNoBindings.RemoveBinding(b)
	}
	for _, b := range clusterBindings {
		storeWithNoBindings.RemoveClusterBinding(b)
	}
	for _, tc := range testCases {
		name := fmt.Sprintf("%q in namespace %q should have %q permision level",
			tc.deployment.GetServiceAccount(), tc.deployment.GetNamespace(), tc.expected)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected.String(), store.GetPermissionLevelForDeployment(tc.deployment).String())
		})

		name = fmt.Sprintf("%q in namespace %q should have NO permisions after removing roles but keeping bindings",
			tc.deployment.GetServiceAccount(), tc.deployment.GetNamespace())
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, storage.PermissionLevel_NONE.String(), storeWithNoRoles.GetPermissionLevelForDeployment(tc.deployment).String())
		})

		name = fmt.Sprintf("%q in namespace %q should have NO permisions after removing bindings but keeping roles",
			tc.deployment.GetServiceAccount(), tc.deployment.GetNamespace())
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, storage.PermissionLevel_NONE.String(), storeWithNoBindings.GetPermissionLevelForDeployment(tc.deployment).String())
		})
	}
}

func role(name string) v1.RoleRef {
	return roleRef(name, "Role")
}

func clusterRole(name string) v1.RoleRef {
	return roleRef(name, "ClusterRole")
}

func roleRef(name, kind string) v1.RoleRef {
	return v1.RoleRef{
		Name: name, Kind: kind, APIGroup: "rbac.authorization.k8s.io",
	}
}

func meta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: name, UID: types.UID(name + "-id"), Namespace: "n1",
	}
}

func setupStore(roles []*v1.Role, clusterRoles []*v1.ClusterRole, bindings []*v1.RoleBinding, clusterBindings []*v1.ClusterRoleBinding) Store {
	tested := NewStore()
	for _, r := range roles {
		tested.UpsertRole(r)
	}
	for _, b := range bindings {
		tested.UpsertBinding(b)
	}
	for _, r := range clusterRoles {
		tested.UpsertClusterRole(r)
	}
	for _, b := range clusterBindings {
		tested.UpsertClusterBinding(b)
	}
	return tested
}
