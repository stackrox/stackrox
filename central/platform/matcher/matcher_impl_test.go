package matcher

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestPlatformMatcher(t *testing.T) {
	suite.Run(t, new(platformMatcherTestSuite))
}

type platformMatcherTestSuite struct {
	suite.Suite

	matcher PlatformMatcher
}

func (s *platformMatcherTestSuite) SetupSuite() {
	s.matcher = New()
}

func (s *platformMatcherTestSuite) TestMatchAlert() {
	// case: nil alert
	match, err := s.matcher.MatchAlert(nil)
	s.Require().Error(err)
	s.Require().False(match)

	// case: alert without embedded deployment
	match, err = s.matcher.MatchAlert(&storage.Alert{EntityType: storage.Alert_DEPLOYMENT})
	s.Require().Error(err)
	s.Require().False(match)

	// case: Alert on a non deployment entity
	alert := &storage.Alert{
		EntityType: storage.Alert_RESOURCE,
		Entity:     &storage.Alert_Resource_{Resource: &storage.Alert_Resource{Name: "dummy_secret"}},
	}
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: Alert on a deployment not matching platform rules
	alert = &storage.Alert{
		EntityType: storage.Alert_DEPLOYMENT,
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Name:        "dep1",
				Namespace:   "my-namespace",
				ClusterName: "cluster1",
			},
		},
	}
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	alert.GetDeployment().Namespace = "openshift-operators"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: Alert on a deployment matching platform rules
	alert.GetDeployment().Namespace = "openshift123"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().Namespace = "kube123"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().Namespace = "redhat123"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)
}

func (s *platformMatcherTestSuite) TestMatchDeployment() {
	// case: nil deployment
	match, err := s.matcher.MatchDeployment(nil)
	s.Require().Error(err)
	s.Require().False(match)

	// case: deployment not matching platform rules
	dep := &storage.Deployment{
		Name:        "dep1",
		Namespace:   "my-namespace",
		ClusterName: "cluster-1",
	}
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	dep.Namespace = "openshift-operators"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: deployment matching platform rules
	dep.Namespace = "redhat123"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "istio-system"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)
}

