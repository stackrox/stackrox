package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/image/policies"
	"bitbucket.org/stack-rox/apollo/pkg/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createBolt() (*BoltDB, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	db, err := New(tmpDir)
	return db, err
}

func TestGetDefaultPolicies(t *testing.T) {
	db, err := createBolt()
	require.NoError(t, err)
	defer db.Close()
	defer os.Remove(db.Path())

	defaults.PoliciesPath = policies.Directory()
	policies, err := defaults.Policies()
	require.NoError(t, err)
	require.NotNil(t, policies)

	for _, p := range policies {
		_, err := matcher.New(p)
		assert.NoError(t, err)
	}
}
