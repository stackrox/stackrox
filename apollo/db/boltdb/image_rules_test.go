package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltImageRules(t *testing.T) {
	suite.Run(t, new(BoltImageRulesTestSuite))
}

type BoltImageRulesTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltImageRulesTestSuite) SetupSuite() {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := MakeBoltDB(tmpDir)
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db.(*BoltDB)
}

func (suite *BoltImageRulesTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltImageRulesTestSuite) TestImageRules() {
	rule1 := &v1.ImageRule{
		Name:     "rule1",
		Severity: v1.Severity_LOW_SEVERITY,
	}
	err := suite.AddImageRule(rule1)
	suite.Nil(err)

	rule2 := &v1.ImageRule{
		Name:     "rule2",
		Severity: v1.Severity_HIGH_SEVERITY,
	}
	err = suite.AddImageRule(rule2)
	suite.Nil(err)
	// Get all alerts
	rules, err := suite.GetImageRules(&v1.GetImageRulesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImageRule{rule1, rule2}, rules)

	rule1.Severity = v1.Severity_HIGH_SEVERITY
	err = suite.UpdateImageRule(rule1)
	suite.Nil(err)
	rules, err = suite.GetImageRules(&v1.GetImageRulesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImageRule{rule1, rule2}, rules)

	err = suite.RemoveImageRule(rule1.Name)
	suite.Nil(err)
	rules, err = suite.GetImageRules(&v1.GetImageRulesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImageRule{rule2}, rules)
}
