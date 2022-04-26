package util

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func noNetworkFieldsPolicy() *storage.Policy {
	return &storage.Policy{
		Id:          "policy",
		Name:        "policy name",
		Description: "policy description",
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: augmentedobjs.ImageScanCustomTag,
					},
				},
			},
		},
	}
}

func networkFieldsPolicy() *storage.Policy {
	return &storage.Policy{
		Id:          "policy",
		Name:        "policy name",
		Description: "policy description",
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: augmentedobjs.MissingIngressPolicyCustomTag,
					},
				},
			},
		},
	}
}

func testAlertsWithoutNetworkPolicyFields() []*storage.Alert {
	return []*storage.Alert{
		{
			Policy: noNetworkFieldsPolicy(),
		},
	}
}

func testAlertsWithNetworkPolicyFields() []*storage.Alert {
	return []*storage.Alert{
		{
			Policy: noNetworkFieldsPolicy(),
		},
		{
			Policy: networkFieldsPolicy(),
		},
	}
}

type alertTestSuite struct {
	suite.Suite
}

func (s *alertTestSuite) TestRemoveAlertWithNetworkPolicyFields() {
	cases := map[string]struct {
		alerts                      []*storage.Alert
		expectedAlerts              []*storage.Alert
		expectedNetworkPolicyFields bool
	}{
		"Alerts without network policy fields": {
			alerts:                      testAlertsWithoutNetworkPolicyFields(),
			expectedAlerts:              testAlertsWithoutNetworkPolicyFields(),
			expectedNetworkPolicyFields: false,
		},
		"Alerts with network policy fields": {
			alerts:                      testAlertsWithNetworkPolicyFields(),
			expectedAlerts:              testAlertsWithoutNetworkPolicyFields(),
			expectedNetworkPolicyFields: true,
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			alerts, has := RemoveAlertsWithNetworkPolicyFields(c.alerts)
			assert.Equal(s.T(), c.expectedAlerts, alerts)
			assert.Equal(s.T(), c.expectedNetworkPolicyFields, has)
		})
	}
}

func TestAlert(t *testing.T) {
	suite.Run(t, new(alertTestSuite))
}
