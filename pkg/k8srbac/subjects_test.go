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
			rb:     &storage.K8SRoleBinding{Subjects: make([]*storage.Subject, 0)},
			expect: nil,
		},
		"all kinds supported": {
			rb: &storage.K8SRoleBinding{Subjects: []*storage.Subject{
				{Kind: storage.SubjectKind_UNSET_KIND},
				{Kind: storage.SubjectKind_SERVICE_ACCOUNT},
				{Kind: storage.SubjectKind_USER},
				{Kind: storage.SubjectKind_GROUP},
			}},
			expect: []*storage.Subject{
				{Kind: storage.SubjectKind_UNSET_KIND},
				{Kind: storage.SubjectKind_SERVICE_ACCOUNT},
				{Kind: storage.SubjectKind_USER},
				{Kind: storage.SubjectKind_GROUP},
			},
		},
		"namespaced kinds preserve namespace": {
			rb: &storage.K8SRoleBinding{Subjects: []*storage.Subject{
				{Kind: storage.SubjectKind_UNSET_KIND, Namespace: "namespace"},
				{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Namespace: "namespace"},
			}},
			expect: []*storage.Subject{
				{Kind: storage.SubjectKind_UNSET_KIND, Namespace: "namespace"},
				{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Namespace: "namespace"},
			},
		},
		"non-namespaced kinds are adjusted": {
			rb: &storage.K8SRoleBinding{Subjects: []*storage.Subject{
				{Kind: storage.SubjectKind_USER, Namespace: "namespace"},
				{Kind: storage.SubjectKind_GROUP, Namespace: "namespace"},
			}},
			expect: []*storage.Subject{
				{Kind: storage.SubjectKind_USER},
				{Kind: storage.SubjectKind_GROUP},
			},
		},
		"only non-namespaced kinds are adjusted for list of mixed kinds": {
			rb: &storage.K8SRoleBinding{Subjects: []*storage.Subject{
				{Kind: storage.SubjectKind_UNSET_KIND, Namespace: "namespace"},
				{Kind: storage.SubjectKind_UNSET_KIND},
				{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Namespace: "namespace"},
				{Kind: storage.SubjectKind_USER, Namespace: "namespace"},
				{Kind: storage.SubjectKind_GROUP, Namespace: "namespace"},
			}},
			expect: []*storage.Subject{
				{Kind: storage.SubjectKind_UNSET_KIND, Namespace: "namespace"},
				{Kind: storage.SubjectKind_UNSET_KIND},
				{Kind: storage.SubjectKind_SERVICE_ACCOUNT, Namespace: "namespace"},
				{Kind: storage.SubjectKind_USER},
				{Kind: storage.SubjectKind_GROUP},
			},
		},
		"non-namespaced duplicates are adjusted": {
			rb: &storage.K8SRoleBinding{Subjects: []*storage.Subject{
				{Kind: storage.SubjectKind_USER, Namespace: "is-first-in-the-list"},
				{Kind: storage.SubjectKind_USER},
				{Kind: storage.SubjectKind_GROUP},
				{Kind: storage.SubjectKind_GROUP, Namespace: "is-second-in-the-list"},
			}},
			expect: []*storage.Subject{
				{Kind: storage.SubjectKind_USER},
				{Kind: storage.SubjectKind_GROUP},
			},
		},
		"only namespace removed": {
			rb: &storage.K8SRoleBinding{Subjects: []*storage.Subject{
				{Kind: storage.SubjectKind_USER, Name: "user-1", Namespace: "namespace", Id: "cluster-1:user-1", ClusterId: "cluster-1", ClusterName: "cluster-name"},
				{Kind: storage.SubjectKind_GROUP, Name: "group-1", Namespace: "namespace", Id: "cluster-1:group-1", ClusterId: "cluster-1", ClusterName: "cluster-name"},
			}},
			expect: []*storage.Subject{
				{Kind: storage.SubjectKind_USER, Name: "user-1", Id: "cluster-1:user-1", ClusterId: "cluster-1", ClusterName: "cluster-name"},
				{Kind: storage.SubjectKind_GROUP, Name: "group-1", Id: "cluster-1:group-1", ClusterId: "cluster-1", ClusterName: "cluster-name"},
			},
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			protoassert.ElementsMatch(t, tt.expect, GetSubjectsAdjustedByKind(tt.rb))
		})
	}
}
