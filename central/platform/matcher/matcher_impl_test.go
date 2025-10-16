package matcher

import (
	"regexp"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func TestPlatformMatcher(t *testing.T) {
	suite.Run(t, new(platformMatcherTestSuite))
}

type platformMatcherTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	matcher  PlatformMatcher
}

func (s *platformMatcherTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
	s.matcher = GetTestPlatformMatcherWithDefaultPlatformComponentConfig(s.mockCtrl)
}

func (s *platformMatcherTestSuite) TestMatchAlert() {
	s.matcher = GetTestPlatformMatcherWithDefaultPlatformComponentConfig(s.mockCtrl)
	// case: nil alert
	match, err := s.matcher.MatchAlert(nil)
	s.Require().Error(err)
	s.Require().False(match)

	// case: alert without embedded deployment
	match, err = s.matcher.MatchAlert(&storage.Alert{})
	s.Require().NoError(err)
	s.Require().False(match)

	// case: Alert on a non deployment entity
	ar := &storage.Alert_Resource{}
	ar.SetName("dummy_secret")
	alert := &storage.Alert{}
	alert.SetResource(proto.ValueOrDefault(ar))
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: Alert on a deployment not matching platform rules
	ad := &storage.Alert_Deployment{}
	ad.SetName("dep1")
	ad.SetNamespace("my-namespace")
	ad.SetClusterName("cluster1")
	alert = &storage.Alert{}
	alert.SetDeployment(proto.ValueOrDefault(ad))
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	alert.GetDeployment().SetNamespace("aap-suffix")
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	alert.GetDeployment().SetNamespace("prefix-hive")
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	alert.GetDeployment().SetNamespace("prefix-openshift-123")
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: Alert on a deployment matching platform rules
	alert.GetDeployment().SetNamespace("openshift-123")
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().SetNamespace("kube-123")
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().SetNamespace("stackrox")
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().SetNamespace("rhacs-operator")
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().SetNamespace("open-cluster-management")
	match, err = s.matcher.MatchAlert(alert)
	s.Require().NoError(err)
	s.Require().True(match)

	alert.GetDeployment().SetNamespace("multicluster-engine")
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
	dep := &storage.Deployment{}
	dep.SetName("dep1")
	dep.SetNamespace("my-namespace")
	dep.SetClusterName("cluster-1")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	dep.SetNamespace("open-cluster-management-suffix")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	dep.SetNamespace("prefix-multicluster-engine")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	dep.SetNamespace("openshift123")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	// case: deployment matching platform rules
	dep.SetNamespace("openshift-123")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("kube-123")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("aap")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("hive")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("stackrox")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("rhacs-operator")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("open-cluster-management")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("nvidia-gpu-operator")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)
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
	dep := &storage.Deployment{}
	dep.SetName("dep1")
	dep.SetNamespace("my-namespace")
	dep.SetClusterName("cluster-1")
	match, err := s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)

	dep.SetNamespace("kube-system")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("openshift")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("bad-namespace")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().True(match)

	dep.SetNamespace("happy-namespace")
	match, err = s.matcher.MatchDeployment(dep)
	s.Require().NoError(err)
	s.Require().False(match)
}
