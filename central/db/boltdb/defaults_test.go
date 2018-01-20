package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/image/policies"
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

	defaultPoliciesPath = policies.Directory()

	policies, err := db.getDefaultPolicies()
	require.NoError(t, err)
	require.NotNil(t, policies)
}
