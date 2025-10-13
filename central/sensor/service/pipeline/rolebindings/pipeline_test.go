package rolebindings

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/protoassert"
)

func Test_enrichSubjects(t *testing.T) {
	clusterId := "cluster-id-1"
	clusterName := "cluster-name-1"

	tests := map[string]struct {
		binding *storage.K8SRoleBinding
		expect  *storage.K8SRoleBinding
	}{
		"nil rolebinding": {
			binding: nil,
			expect:  nil,
		},
		"nil subjects": {
			binding: &storage.K8SRoleBinding{},
			expect:  &storage.K8SRoleBinding{},
		},
		"empty subjects": {
			binding: &storage.K8SRoleBinding{Subjects: make([]*storage.Subject, 0)},
			expect:  &storage.K8SRoleBinding{Subjects: make([]*storage.Subject, 0)},
		},
		"all rolebinding kinds": {
			binding: &storage.K8SRoleBinding{
				ClusterId:   clusterId,
				ClusterName: clusterName,
				Subjects: []*storage.Subject{
					{Kind: storage.SubjectKind_UNSET_KIND, Name: "sub-1"},
					{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Name: "sub-2"},
					{Kind: storage.SubjectKind_USER, Name: "sub-3"},
					{Kind: storage.SubjectKind_GROUP, Name: "sub-4"},
				},
			},
			expect: &storage.K8SRoleBinding{
				ClusterId:   clusterId,
				ClusterName: clusterName,
				Subjects: []*storage.Subject{
					{
						Id:          k8srbac.CreateSubjectID(clusterId, "sub-1"),
						Kind:        storage.SubjectKind_UNSET_KIND,
						Name:        "sub-1",
						ClusterId:   clusterId,
						ClusterName: clusterName,
					},
					{
						Id:          k8srbac.CreateSubjectID(clusterId, "sub-2"),
						Kind:        storage.SubjectKind_SERVICE_ACCOUNT,
						Name:        "sub-2",
						ClusterId:   clusterId,
						ClusterName: clusterName,
					},
					{
						Id:          k8srbac.CreateSubjectID(clusterId, "sub-3"),
						Kind:        storage.SubjectKind_USER,
						Name:        "sub-3",
						ClusterId:   clusterId,
						ClusterName: clusterName,
					},
					{
						Id:          k8srbac.CreateSubjectID(clusterId, "sub-4"),
						Kind:        storage.SubjectKind_GROUP,
						Name:        "sub-4",
						ClusterId:   clusterId,
						ClusterName: clusterName,
					},
				},
			},
		},
		"all rolebinding kinds with namespace": {
			binding: &storage.K8SRoleBinding{
				ClusterId:   clusterId,
				ClusterName: clusterName,
				Subjects: []*storage.Subject{
					{Kind: storage.SubjectKind_UNSET_KIND, Name: "sub-1", Namespace: "ns-1"},
					{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Name: "sub-2", Namespace: "ns-2"},
					{Kind: storage.SubjectKind_USER, Name: "sub-3", Namespace: "ns-3"},
					{Kind: storage.SubjectKind_GROUP, Name: "sub-4", Namespace: "ns-4"},
				},
			},
			expect: &storage.K8SRoleBinding{
				ClusterId:   clusterId,
				ClusterName: clusterName,
				Subjects: []*storage.Subject{
					{
						Id:          k8srbac.CreateSubjectID(clusterId, "sub-1"),
						Kind:        storage.SubjectKind_UNSET_KIND,
						Name:        "sub-1",
						Namespace:   "ns-1",
						ClusterId:   clusterId,
						ClusterName: clusterName,
					},
					{
						Id:          k8srbac.CreateSubjectID(clusterId, "sub-2"),
						Kind:        storage.SubjectKind_SERVICE_ACCOUNT,
						Name:        "sub-2",
						Namespace:   "ns-2",
						ClusterId:   clusterId,
						ClusterName: clusterName,
					},
					{
						Id:          k8srbac.CreateSubjectID(clusterId, "sub-3"),
						Kind:        storage.SubjectKind_USER,
						Name:        "sub-3",
						Namespace:   "ns-3",
						ClusterId:   clusterId,
						ClusterName: clusterName,
					},
					{
						Id:          k8srbac.CreateSubjectID(clusterId, "sub-4"),
						Kind:        storage.SubjectKind_GROUP,
						Name:        "sub-4",
						Namespace:   "ns-4",
						ClusterId:   clusterId,
						ClusterName: clusterName,
					},
				},
			},
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			enrichSubjects(tt.binding)
			protoassert.Equal(t, tt.expect, tt.binding)
		})
	}
}
