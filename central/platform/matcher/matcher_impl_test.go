package matcher

import (
	"regexp"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPlatformMatcher(t *testing.T) {
	suite.Run(t, new(platformMatcherTestSuite))
}

type platformMatcherTestSuite struct {
	suite.Suite

	matcher PlatformMatcher
}

func (s *platformMatcherTestSuite) SetupSuite() {
	mockCtrl := gomock.NewController(s.T())
	s.matcher = GetTestPlatformMatcherWithDefaultPlatformComponentConfig(mockCtrl)
}

func (s *platformMatcherTestSuite) TestMatchAlert() {
	// case: nil alert
	match, err := s.matcher.MatchAlert(nil)
	s.Require().Error(err)
	s.Require().False(match)

	// case: alert without embedded deployment
	match, err = s.matcher.MatchAlert(&storage.Alert{})
	s.Require().NoError(err)
	s.Require().False(match)

	// case: Alert on a non deployment entity
	alert := &storage.Alert{
		Entity: &storage.Alert_Resource_{Resource: &storage.Alert_Resource{Name: "dummy_secret"}},
	}
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: Alert on a deployment not matching platform rules
	alert = &storage.Alert{
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

	alert.GetDeployment().Namespace = "aap-suffix"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	alert.GetDeployment().Namespace = "prefix-hive"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	alert.GetDeployment().Namespace = "prefix-openshift-123"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: Alert on a deployment matching platform rules
	alert.GetDeployment().Namespace = "openshift-123"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().Namespace = "kube-123"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().Namespace = "stackrox"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().Namespace = "rhacs-operator"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().Namespace = "open-cluster-management"
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().Namespace = "multicluster-engine"
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

	dep.Namespace = "open-cluster-management-suffix"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	dep.Namespace = "prefix-multicluster-engine"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	dep.Namespace = "openshift123"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: deployment matching platform rules
	dep.Namespace = "openshift-123"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "kube-123"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "aap"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "hive"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "stackrox"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "rhacs-operator"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "open-cluster-management"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "nvidia-gpu-operator"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)
}

func (s *platformMatcherTestSuite) TestCustomPlatformComponentRegexes() {
	// Try to enable Customizable Platform Components feature
	if !features.CustomizablePlatformComponents.Enabled() {
		s.T().Setenv(features.CustomizablePlatformComponents.EnvVar(), "true")
	}
	// If we weren't able to set the environment variable for some reason, skip this test
	if !features.CustomizablePlatformComponents.Enabled() {
		s.T().Skip("Customized platform components was not enabled")
	}
	regexes := []*regexp.Regexp{
		regexp.MustCompile("kube.*"),
		regexp.MustCompile("openshift.*"),
		regexp.MustCompile("bad-namespace.*"),
	}
	s.matcher.SetRegexes(regexes)
	dep := &storage.Deployment{
		Name:        "dep1",
		Namespace:   "my-namespace",
		ClusterName: "cluster-1",
	}
	match, err := s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	dep.Namespace = "kube-system"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "openshift"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "bad-namespace"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.Namespace = "happy-namespace"
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)
}
