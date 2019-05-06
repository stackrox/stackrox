package resolvers

import (
	"context"
	"sort"

	"github.com/stackrox/rox/pkg/set"
)

// StringListEntryResolver represents a set of values keyed by a string
type stringListEntryResolver struct {
	key    string
	values set.StringSet
}

// ScopedPermissionsResolver represents the scoped permissions of a subject/service account
type scopedPermissionsResolver struct {
	scope       string
	permissions []*stringListEntryResolver
}

// Key represents the key value of the string list entry
func (resolver *stringListEntryResolver) Key(ctx context.Context) string {
	return resolver.key
}

// Values represents the set of values of the string list entry
func (resolver *stringListEntryResolver) Values(ctx context.Context) []string {
	return resolver.values.AsSlice()
}

func wrapStringListEntries(values map[string]set.StringSet) []*stringListEntryResolver {
	if len(values) == 0 {
		return nil
	}

	output := make([]*stringListEntryResolver, 0, len(values))
	for i, v := range values {
		output = append(output, &stringListEntryResolver{i, v})
	}

	return output
}

// Scope represents the scope of the permissions - cluster wide or the namespace name to which the permissions are scoped
func (resolver *scopedPermissionsResolver) Scope(ctx context.Context) string {
	return resolver.scope
}

// Permissions represents the verbs and the resources to which those verbs are granted
func (resolver *scopedPermissionsResolver) Permissions(ctx context.Context) []*stringListEntryResolver {
	return resolver.permissions
}

// WrapPermissions wraps the input into a scopedPermissionsResolver
func wrapPermissions(values map[string]map[string]set.StringSet) []*scopedPermissionsResolver {
	if len(values) == 0 {
		return nil
	}
	output := make([]*scopedPermissionsResolver, 0, len(values))
	for scope, permissions := range values {
		output = append(output, &scopedPermissionsResolver{scope, wrapStringListEntries(permissions)})
	}

	sort.SliceStable(output, func(i, j int) bool { return output[i].scope < output[j].scope })
	return output
}

type subjectWithClusterIDResolver struct {
	clusterID string
	subject   *subjectResolver
}

func (resolver *subjectWithClusterIDResolver) ClusterID(ctx context.Context) string {
	return resolver.clusterID
}

func (resolver *subjectWithClusterIDResolver) Subject(ctx context.Context) *subjectResolver {
	return resolver.subject
}

func wrapSubjects(clusterID string, subjects []*subjectResolver) []*subjectWithClusterIDResolver {
	if len(subjects) == 0 {
		return nil
	}

	output := make([]*subjectWithClusterIDResolver, 0, len(subjects))
	for _, s := range subjects {
		output = append(output, &subjectWithClusterIDResolver{clusterID, s})
	}

	return output

}

func wrapSubject(clusterID string, subject *subjectResolver) *subjectWithClusterIDResolver {
	return &subjectWithClusterIDResolver{clusterID, subject}
}
