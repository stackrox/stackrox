package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

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

func TestGetDefaultImagePolicies(t *testing.T) {
	db, err := createBolt()
	require.NoError(t, err)
	defer db.Close()
	defer os.Remove(db.Path())

	defaultPoliciesPath = os.Getenv("GOPATH") + "/src/bitbucket.org/stack-rox/apollo/image/policies"

	imagePolicies, err := db.getDefaultImagePolicies()
	require.NoError(t, err)
	require.NotNil(t, imagePolicies)
}
