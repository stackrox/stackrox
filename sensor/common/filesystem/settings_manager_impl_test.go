package filesystem

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

func TestFactSettingsManager(t *testing.T) {
	suite.Run(t, new(FactSettingsManagerSuite))
}

type FactSettingsManagerSuite struct {
	suite.Suite

	mgr *factSettingsManager
}

func (s *FactSettingsManagerSuite) SetupTest() {
	s.mgr = NewFactSettingsManager().(*factSettingsManager)
}

func newTestPolicy(disabled bool, eventSource storage.EventSource, paths ...string) *storage.Policy {
	var pathValues []*storage.PolicyValue
	for _, p := range paths {
		pathValues = append(pathValues, &storage.PolicyValue{Value: p})
	}

	return &storage.Policy{
		Id:              uuid.NewV4().String(),
		PolicyVersion:   "1.1",
		Name:            "Test File Access Policy",
		Disabled:        disabled,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     eventSource,
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.FilePath,
						Values:    pathValues,
					},
				},
			},
		},
	}
}

func (s *FactSettingsManagerSuite) TestExtractPaths() {
	cases := map[string]struct {
		policies []*storage.Policy
		expected []string
	}{
		"no policies": {
			policies: nil,
			expected: nil,
		},
		"single path": {
			policies: []*storage.Policy{
				newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/etc/passwd"),
			},
			expected: []string{"/etc/passwd"},
		},
		"multiple paths": {
			policies: []*storage.Policy{
				newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/etc/passwd", "/etc/shadow"),
			},
			expected: []string{"/etc/passwd", "/etc/shadow"},
		},
		"deduplicates across policies": {
			policies: []*storage.Policy{
				newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/etc/passwd"),
				newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/etc/passwd", "/etc/shadow"),
			},
			expected: []string{"/etc/passwd", "/etc/shadow"},
		},
		"disabled policies excluded": {
			policies: []*storage.Policy{
				newTestPolicy(true, storage.EventSource_DEPLOYMENT_EVENT, "/etc/passwd"),
				newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/etc/shadow"),
			},
			expected: []string{"/etc/shadow"},
		},
		"non-runtime policies excluded": {
			policies: []*storage.Policy{
				{
					Id:              uuid.NewV4().String(),
					PolicyVersion:   "1.1",
					LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
					EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
					PolicySections: []*storage.PolicySection{
						{
							PolicyGroups: []*storage.PolicyGroup{
								{
									FieldName: fieldnames.ImageTag,
									Values:    []*storage.PolicyValue{{Value: "nginx"}},
								},
							},
						},
					},
				},
			},
			expected: nil,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			result := s.mgr.extractFileActivityPaths(tc.policies)
			assert.ElementsMatch(s.T(), tc.expected, result.AsSlice())
		})
	}
}

func (s *FactSettingsManagerSuite) TestUpdateCachesIdenticalPolicies() {
	policies := []*storage.Policy{
		newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/etc/passwd"),
	}

	s.mgr.UpdateFactSettings(policies)
	it := s.mgr.ConfigMapStream().Iterator(false)
	require.NotNil(s.T(), it.Value())

	// Same policies again should not push.
	s.mgr.UpdateFactSettings(policies)
	select {
	case <-it.Done():
		s.T().Fatal("expected no update for identical policies")
	default:
	}
}

func (s *FactSettingsManagerSuite) TestUpdatePushesOnChange() {
	s.mgr.UpdateFactSettings([]*storage.Policy{
		newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/etc/passwd"),
	})
	it := s.mgr.ConfigMapStream().Iterator(false)
	require.NotNil(s.T(), it.Value())

	s.mgr.UpdateFactSettings([]*storage.Policy{
		newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/etc/shadow"),
	})
	<-it.Done()
	it = it.TryNext()
	require.NotNil(s.T(), it.Value())
}

func (s *FactSettingsManagerSuite) TestUpdatePathsSorted() {
	s.mgr.UpdateFactSettings([]*storage.Policy{
		newTestPolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "/z/path", "/a/path", "/m/path"),
	})
	it := s.mgr.ConfigMapStream().Iterator(false)
	cm := it.Value()
	require.NotNil(s.T(), cm)

	var settings sensor.FactSettings
	require.NoError(s.T(), yaml.Unmarshal([]byte(cm.Data[factConfigFile]), &settings))
	assert.Equal(s.T(), []string{"/a/path", "/m/path", "/z/path"}, settings.GetPaths())
}
