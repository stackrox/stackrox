package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
)

func TestGetSubjectsAdjustedByKind(t *testing.T) {
	tests := map[string]struct {
		rb     *storage.K8SRoleBinding
		expect []*storage.Subject
	}{
		"nil rolebinding": {
			rb:     nil,
			expect: nil,
		},
		"nil subjects": {
			rb:     &storage.K8SRoleBinding{},
			expect: nil,
		},
		"empty subjects": {
			rb:     storage.K8SRoleBinding_builder{Subjects: make([]*storage.Subject, 0)}.Build(),
			expect: nil,
		},
		"all kinds supported": {
			rb: storage.K8SRoleBinding_builder{Subjects: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_UNSET_KIND}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_SERVICE_ACCOUNT}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_USER}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP}.Build(),
			}}.Build(),
			expect: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_UNSET_KIND}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_SERVICE_ACCOUNT}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_USER}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP}.Build(),
			},
		},
		"namespaced kinds preserve namespace": {
			rb: storage.K8SRoleBinding_builder{Subjects: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_UNSET_KIND, Namespace: "namespace"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Namespace: "namespace"}.Build(),
			}}.Build(),
			expect: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_UNSET_KIND, Namespace: "namespace"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Namespace: "namespace"}.Build(),
			},
		},
		"non-namespaced kinds are adjusted": {
			rb: storage.K8SRoleBinding_builder{Subjects: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_USER, Namespace: "namespace"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP, Namespace: "namespace"}.Build(),
			}}.Build(),
			expect: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_USER}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP}.Build(),
			},
		},
		"only non-namespaced kinds are adjusted for list of mixed kinds": {
			rb: storage.K8SRoleBinding_builder{Subjects: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_UNSET_KIND, Namespace: "namespace"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_UNSET_KIND}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Namespace: "namespace"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_USER, Namespace: "namespace"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP, Namespace: "namespace"}.Build(),
			}}.Build(),
			expect: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_UNSET_KIND, Namespace: "namespace"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_UNSET_KIND}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Namespace: "namespace"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_USER}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP}.Build(),
			},
		},
		"non-namespaced duplicates are adjusted": {
			rb: storage.K8SRoleBinding_builder{Subjects: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_USER, Namespace: "is-first-in-the-list"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_USER}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP, Namespace: "is-second-in-the-list"}.Build(),
			}}.Build(),
			expect: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_USER}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP}.Build(),
			},
		},
		"only namespace removed": {
			rb: storage.K8SRoleBinding_builder{Subjects: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_USER, Name: "user-1", Namespace: "namespace", Id: "cluster-1:user-1", ClusterId: "cluster-1", ClusterName: "cluster-name"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP, Name: "group-1", Namespace: "namespace", Id: "cluster-1:group-1", ClusterId: "cluster-1", ClusterName: "cluster-name"}.Build(),
			}}.Build(),
			expect: []*storage.Subject{
				storage.Subject_builder{Kind: storage.SubjectKind_USER, Name: "user-1", Id: "cluster-1:user-1", ClusterId: "cluster-1", ClusterName: "cluster-name"}.Build(),
				storage.Subject_builder{Kind: storage.SubjectKind_GROUP, Name: "group-1", Id: "cluster-1:group-1", ClusterId: "cluster-1", ClusterName: "cluster-name"}.Build(),
			},
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			protoassert.ElementsMatch(t, tt.expect, GetSubjectsAdjustedByKind(tt.rb))
		})
	}
}
