package inmem

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/db/boltdb"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createBoltDB() (db.Storage, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, fmt.Errorf("Failed to get temporary directory: %v", err.Error())
	}
	db, err := boltdb.MakeBoltDB(tmpDir)
	if err != nil {
		return nil, err
	}
	return db, err
}

func TestLoad(t *testing.T) {
	persistent, err := createBoltDB()
	require.Nil(t, err)
	bolt := persistent.(*boltdb.BoltDB)
	defer os.Remove(bolt.Path())
	defer persistent.Close()

	inmem := New(persistent)
	image := &v1.Image{
		Sha: "sha",
	}
	persistent.AddImage(image)
	alert := &v1.Alert{
		Id: "id1",
	}
	persistent.AddAlert(alert)
	imageRule := &v1.ImageRule{
		Name: "rule1",
	}
	persistent.AddImageRule(imageRule)
	inmem.Load()

	assert.Equal(t, map[string]*v1.Image{image.Sha: image}, inmem.images)
	assert.Equal(t, map[string]*v1.Alert{alert.Id: alert}, inmem.alerts)
	assert.Equal(t, map[string]*v1.ImageRule{imageRule.Name: imageRule}, inmem.imageRules)
}
