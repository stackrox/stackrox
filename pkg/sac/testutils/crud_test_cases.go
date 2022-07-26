package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/sac"
)

// Verbs for the Generic*SACUpsertTestCases functions.
const (
	VerbAdd    = "add"
	VerbUpdate = "update"
	VerbUpsert = "upsert"
)

// SACCrudTestCase is used within SAC tests. It describes the expected behaviour of a datastore CRUD function for
// a given scoped context. The contexts are defined in the current package in the test_contexts.go file and
// are referred to by their key in test cases.
type SACCrudTestCase struct {
	ScopeKey      string
	ExpectedError error
	ExpectError   bool
	ExpectedFound bool
}

// GenericNamespaceSACUpsertTestCases returns a generic set of SACCrudTestCase.
// It is appropriate for use in the context of testing Upsert function on namespace scoped resources
// when the scope checks are expected to assess whether the object to upsert belongs to a namespace
// in scope. These test cases assume the inserted test object belongs to Cluster2 and NamespaceB.
func GenericNamespaceSACUpsertTestCases(_ *testing.T, verb string) map[string]SACCrudTestCase {
	return map[string]SACCrudTestCase{
		"(full) read-only cannot " + verb: {
			ScopeKey:      UnrestrictedReadCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"full read-write can " + verb: {
			ScopeKey:      UnrestrictedReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
		"full read-write on wrong cluster cannot " + verb: {
			ScopeKey:      Cluster1ReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and wrong namespace name cannot " + verb: {
			ScopeKey:      Cluster1NamespaceAReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace name cannot " + verb: {
			ScopeKey:      Cluster1NamespaceBReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"full read-write on right cluster can " + verb: {
			ScopeKey:      Cluster2ReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
		"read-write on right cluster but wrong namespace cannot " + verb: {
			ScopeKey:      Cluster2NamespaceAReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on right cluster but wrong namespaces cannot " + verb: {
			ScopeKey:      Cluster2NamespacesACReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on the right cluster and namespace can " + verb: {
			ScopeKey:      Cluster2NamespaceBReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
		"read-write on the right cluster and at least the right namespace can " + verb: {
			ScopeKey:      Cluster2NamespacesABReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
	}
}

// GenericNamespaceSACGetTestCases returns a generic set of SACCrudTestCase.
// It is appropriate for use in the context of testing Get function on namespace scoped resources
// when the scope checks are expected to assess whether the retrieved object belongs to a namespace
// in scope. These test cases assume the tested object belongs to Cluster2 and NamespaceB.
func GenericNamespaceSACGetTestCases(_ *testing.T) map[string]SACCrudTestCase {
	return map[string]SACCrudTestCase{
		"(full) read-only can get": {
			ScopeKey:      UnrestrictedReadCtx,
			ExpectedFound: true,
		},
		"full read-write can get": {
			ScopeKey:      UnrestrictedReadWriteCtx,
			ExpectedFound: true,
		},
		"full read-write on wrong cluster cannot get": {
			ScopeKey:      Cluster1ReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on wrong cluster and wrong namespace name cannot get": {
			ScopeKey:      Cluster1NamespaceAReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on wrong cluster and matching namespace name cannot get": {
			ScopeKey:      Cluster1NamespaceBReadWriteCtx,
			ExpectedFound: false,
		},
		"full read-write on right cluster can get": {
			ScopeKey:      Cluster2ReadWriteCtx,
			ExpectedFound: true,
		},
		"read-write on right cluster but wrong namespace cannot get": {
			ScopeKey:      Cluster2NamespaceAReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on right cluster but wrong namespaces cannot get": {
			ScopeKey:      Cluster2NamespacesACReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on the right cluster and namespace can get": {
			ScopeKey:      Cluster2NamespaceBReadWriteCtx,
			ExpectedFound: true,
		},
		"read-write on the right cluster and at least the right namespace can get": {
			ScopeKey:      Cluster2NamespacesABReadWriteCtx,
			ExpectedFound: true,
		},
	}
}

// GenericNamespaceSACDeleteTestCases returns a generic set of SACCrudTestCase.
// It is appropriate for use in the context of testing Delete or Remove function on resources when the scope checks
// are expected to assess whether the retrieved object belongs to a namespace
// in scope. These test cases assume the removed test object belongs to Cluster2 and NamespaceB.
func GenericNamespaceSACDeleteTestCases(_ *testing.T) map[string]SACCrudTestCase {
	return map[string]SACCrudTestCase{
		"global read-only should not be able to delete": {
			ScopeKey:      UnrestrictedReadCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"global read-write should be able to delete": {
			ScopeKey:      UnrestrictedReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
		"read-write on wrong cluster should not be able to delete": {
			ScopeKey:      Cluster1ReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and namespace should not be able to delete": {
			ScopeKey:      Cluster1NamespaceAReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace should not be able to delete": {
			ScopeKey:      Cluster1NamespaceBReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster should be able to delete": {
			ScopeKey:      Cluster2ReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
		"read-write on matching cluster and wrong namespace should not be able to delete": {
			ScopeKey:      Cluster2NamespaceAReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and wrong namespaces should not be able to delete": {
			ScopeKey:      Cluster2NamespacesACReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and matching namespace should be able to delete": {
			ScopeKey:      Cluster2NamespaceBReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
		"read-write on matching cluster and at least one matching namespace should be able to delete": {
			ScopeKey:      Cluster2NamespacesABReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
	}
}

// GenericGlobalSACUpsertTestCases returns a generic set of SACCrudTestCase.
// It is appropriate for use in the context of testing Upsert function on resources when the scope checks
// are expected to check global resource access only. These test cases assume the inserted test object
// belongs to Cluster2 and NamespaceB.
func GenericGlobalSACUpsertTestCases(_ *testing.T, verb string) map[string]SACCrudTestCase {
	return map[string]SACCrudTestCase{
		"global read-only should not be able to " + verb: {
			ScopeKey:      UnrestrictedReadCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"global read-write should be able to " + verb: {
			ScopeKey:      UnrestrictedReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
		"read-write on wrong cluster should not be able to " + verb: {
			ScopeKey:      Cluster1ReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and namespace should not be able to " + verb: {
			ScopeKey:      Cluster1NamespaceAReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace should not be able to " + verb: {
			ScopeKey:      Cluster1NamespaceBReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and wrong namespace should not be able to " + verb: {
			ScopeKey:      Cluster2NamespaceAReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and wrong namespaces should not be able to " + verb: {
			ScopeKey:      Cluster2NamespacesACReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and matching namespace should be able to " + verb: {
			ScopeKey:      Cluster2NamespaceBReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and at least one matching namespace should be able to " + verb: {
			ScopeKey:      Cluster2NamespacesABReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
	}
}

// GenericGlobalSACGetTestCases returns a generic set of SACCrudTestCase.
// It is appropriate for use in the context of testing Get function on namespace scoped resources
// when the scope checks are expected to check global resource access only.
// These test cases assume the tested object belongs to Cluster2 and NamespaceB.
func GenericGlobalSACGetTestCases(_ *testing.T) map[string]SACCrudTestCase {
	return map[string]SACCrudTestCase{
		"(full) read-only can get": {
			ScopeKey:      UnrestrictedReadCtx,
			ExpectedFound: true,
		},
		"full read-write can get": {
			ScopeKey:      UnrestrictedReadWriteCtx,
			ExpectedFound: true,
		},
		"full read-write on wrong cluster cannot get": {
			ScopeKey:      Cluster1ReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on wrong cluster and wrong namespace name cannot get": {
			ScopeKey:      Cluster1NamespaceAReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on wrong cluster and matching namespace name cannot get": {
			ScopeKey:      Cluster1NamespaceBReadWriteCtx,
			ExpectedFound: false,
		},
		"full read-write on right cluster cannot get": {
			ScopeKey:      Cluster2ReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on right cluster but wrong namespace cannot get": {
			ScopeKey:      Cluster2NamespaceAReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on right cluster but wrong namespaces cannot get": {
			ScopeKey:      Cluster2NamespacesACReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on the right cluster and namespace cannot get": {
			ScopeKey:      Cluster2NamespaceBReadWriteCtx,
			ExpectedFound: false,
		},
		"read-write on the right cluster and at least the right namespace cannot get": {
			ScopeKey:      Cluster2NamespacesABReadWriteCtx,
			ExpectedFound: false,
		},
	}
}

// GenericGlobalSACDeleteTestCases returns a generic set of SACCrudTestCase.
// It is appropriate for use in the context of testing Delete or Remove function on resources when the scope checks
// are expected to check global resource access only. These test cases assume the removed test object
// belongs to Cluster2 and NamespaceB.
func GenericGlobalSACDeleteTestCases(_ *testing.T) map[string]SACCrudTestCase {
	return map[string]SACCrudTestCase{
		"global read-only should not be able to delete": {
			ScopeKey:      UnrestrictedReadCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"global read-write should be able to delete": {
			ScopeKey:      UnrestrictedReadWriteCtx,
			ExpectError:   false,
			ExpectedError: nil,
		},
		"read-write on wrong cluster should not be able to delete": {
			ScopeKey:      Cluster1ReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and namespace should not be able to delete": {
			ScopeKey:      Cluster1NamespaceAReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace should not be able to delete": {
			ScopeKey:      Cluster1NamespaceBReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster should not be able to delete": {
			ScopeKey:      Cluster2ReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and wrong namespace should not be able to delete": {
			ScopeKey:      Cluster2NamespaceAReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and wrong namespaces should not be able to delete": {
			ScopeKey:      Cluster2NamespacesACReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and matching namespace not should be able to delete": {
			ScopeKey:      Cluster2NamespaceBReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and at least one matching namespace should not be able to delete": {
			ScopeKey:      Cluster2NamespacesABReadWriteCtx,
			ExpectError:   true,
			ExpectedError: sac.ErrResourceAccessDenied,
		},
	}
}
