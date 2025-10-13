package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSystemNamespace(t *testing.T) {
	// OpenShift console is confusing in that if an operator is installed for "all namespaces", clicking on it
	// post installation will take the user to the "openshift-operators" project. Then, if they click on one of the
	// CRs to create it, it will attempt to create those in the "openshift-operators" namespace.
	// There are a bunch of namespaces where the operator shouldn't be deployed in as well, but the ones below are
	// high risk and we want to make sure we avoid them.
	assert.True(t, IsSystemNamespace("openshift-operators"))
	assert.True(t, IsSystemNamespace("openshift-marketplace"))

	assert.False(t, IsSystemNamespace("stackrox"))
}
