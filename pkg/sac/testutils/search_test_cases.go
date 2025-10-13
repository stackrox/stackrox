package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/sac/testconsts"
)

// SACSearchTestCase is used within SAC tests. It will yield the number of found
// items per cluster & namespace.
type SACSearchTestCase struct {
	ScopeKey string
	Results  map[string]map[string]int
}

// GenericScopedSACSearchTestCases returns a generic set of SACSearchTestCase.
// It is appropriate to use when the store contains:
// 9 objects scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 objects scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 objects scoped to Cluster3, 3 to each Namespace A / B / C.
func GenericScopedSACSearchTestCases(_ *testing.T) map[string]SACSearchTestCase {
	return map[string]SACSearchTestCase{
		"Cluster1 read-write access should only see Cluster1 objects": {
			ScopeKey: Cluster1ReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
			},
		},
		"Cluster1 and NamespaceA read-write access should only see Cluster1 and NamespaceA objects": {
			ScopeKey: Cluster1NamespaceAReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
				},
			},
		},
		"Cluster1 and NamespaceB read-write access should only see Cluster1 and NamespaceB objects": {
			ScopeKey: Cluster1NamespaceBReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceB: 3,
				},
			},
		},
		"Cluster1 and NamespaceC read-write access should only see Cluster1 and NamespaceB objects": {
			ScopeKey: Cluster1NamespaceCReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceC: 3,
				},
			},
		},
		"Cluster1 and Namespaces A and B read-write access should only see appropriate cluster/namespace " +
			"objects": {
			ScopeKey: Cluster1NamespacesABReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
				},
			},
		},
		"Cluster1 and Namespaces A and C read-write access should only see appropriate cluster/namespace " +
			"objects": {
			ScopeKey: Cluster1NamespacesACReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceC: 3,
				},
			},
		},
		"Cluster1 and Namespaces B and C read-write access should only see appropriate cluster/namespace " +
			"objects": {
			ScopeKey: Cluster1NamespacesBCReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
			},
		},
		"Cluster2 read-write access should only see Cluster2 objects": {
			ScopeKey: Cluster2ReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
			},
		},
		"Cluster2 and NamespaceA read-write access should see Cluster2 and NamespaceA objects": {
			ScopeKey: Cluster2NamespaceAReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceA: 3,
				},
			},
		},
		"Cluster2 and NamespaceB read-write access should only see Cluster2 and NamespaceB objects": {
			ScopeKey: Cluster2NamespaceBReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceB: 3,
				},
			},
		},
		"Cluster2 and NamespaceC read-write access should only see Cluster2 and NamespaceC objects": {
			ScopeKey: Cluster2NamespaceCReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceC: 3,
				},
			},
		},
		"Cluster2 and Namespaces A and B read-write access should only see appropriate cluster/namespace " +
			"objects": {
			ScopeKey: Cluster2NamespacesABReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
				},
			},
		},
		"Cluster2 and Namespaces A and C read-write access should only see appropriate cluster/namespace " +
			"objects": {
			ScopeKey: Cluster2NamespacesACReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceC: 3,
				},
			},
		},
		"Cluster2 and Namespaces B and C read-write access should only see appropriate cluster/namespace " +
			"objects": {
			ScopeKey: Cluster2NamespacesBCReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
			},
		},
	}
}

// GenericUnrestrictedRawSACSearchTestCases returns a generic set of SACSearchTestCase.
// It is appropriate to use when the store contains:
// 9 objects scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 objects scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 objects scoped to Cluster3, 3 to each Namespace A / B / C.
func GenericUnrestrictedRawSACSearchTestCases(_ *testing.T) map[string]SACSearchTestCase {
	return map[string]SACSearchTestCase{
		"global read access should see all objects": {
			ScopeKey: UnrestrictedReadCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
				testconsts.Cluster2: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
				testconsts.Cluster3: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
			},
		},
		"global read-write access should see all objects": {
			ScopeKey: UnrestrictedReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
				testconsts.Cluster2: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
				testconsts.Cluster3: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 3,
					testconsts.NamespaceC: 3,
				},
			},
		},
	}
}
