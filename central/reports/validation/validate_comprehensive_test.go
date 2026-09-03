package validation

import (
	"testing"

	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	reportConfigDSMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	snapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	collectionDSMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestValidateReportConfiguration(t *testing.T) {
	t.Setenv(features.VulnerabilityReportsEnhancedFiltering.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
	collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
	reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
	snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)

	validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

	tests := map[string]struct {
		config      *apiV2.ReportConfiguration
		setupMocks  func()
		expectError bool
		errContains string
	}{
		"empty name fails": {
			config: &apiV2.ReportConfiguration{
				Name: "",
			},
			setupMocks:  func() {},
			expectError: true,
			errContains: "name is empty",
		},
		"scheduled report without notifiers fails": {
			config: &apiV2.ReportConfiguration{
				Name: "Test Report",
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_DAILY,
					Hour:         9,
					Minute:       0,
				},
			},
			setupMocks:  func() {},
			expectError: true,
			errContains: "must specify a notifier",
		},
		"missing resource scope fails": {
			config: &apiV2.ReportConfiguration{
				Name: "Test Report",
			},
			setupMocks:  func() {},
			expectError: true,
			errContains: "must specify a valid resource scope",
		},
		"valid node report with entity scope": {
			config: &apiV2.ReportConfiguration{
				Name: "Node Report",
				Type: apiV2.ReportConfiguration_NODE_VULNERABILITY,
				ResourceScope: &apiV2.ResourceScope{
					ScopeReference: &apiV2.ResourceScope_EntityScope{
						EntityScope: &apiV2.EntityScope{
							Rules: []*apiV2.EntityScopeRule{
								{
									Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
									Field:  apiV2.ScopeField_FIELD_NAME,
									Values: []*apiV2.RuleValue{
										{Value: "prod", MatchType: apiV2.MatchType_EXACT},
									},
								},
							},
						},
					},
				},
				Filter: &apiV2.ReportConfiguration_NodeVulnReportFilters{
					NodeVulnReportFilters: &apiV2.NodeVulnerabilityReportFilters{
						CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
							AllVuln: true,
						},
					},
				},
			},
			setupMocks:  func() {},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setupMocks()
			err := validator.ValidateReportConfiguration(tc.config)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNotifiers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
	collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
	reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
	snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)

	validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

	tests := map[string]struct {
		config      *apiV2.ReportConfiguration
		setupMocks  func()
		expectError bool
		errContains string
	}{
		"no notifiers without schedule is valid": {
			config: &apiV2.ReportConfiguration{
				Notifiers: nil,
				Schedule:  nil,
			},
			setupMocks:  func() {},
			expectError: false,
		},
		"notifier without email config fails": {
			config: &apiV2.ReportConfiguration{
				Notifiers: []*apiV2.NotifierConfiguration{
					{},
				},
			},
			setupMocks:  func() {},
			expectError: true,
			errContains: "must specify an email notifier configuration",
		},
		"valid notifier": {
			config: &apiV2.ReportConfiguration{
				Notifiers: []*apiV2.NotifierConfiguration{
					{
						NotifierConfig: &apiV2.NotifierConfiguration_EmailConfig{
							EmailConfig: &apiV2.EmailNotifierConfiguration{
								NotifierId:   "notifier-1",
								MailingLists: []string{"user@example.com"},
							},
						},
					},
				},
			},
			setupMocks: func() {
				notifierDS.EXPECT().Exists(gomock.Any(), "notifier-1").Return(true, nil)
			},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setupMocks()
			err := validator.validateNotifiers(tc.config)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmailConfig(t *testing.T) {
	tests := map[string]struct {
		emailConfig *apiV2.EmailNotifierConfiguration
		setupMocks  func(*notifierDSMocks.MockDataStore)
		expectError bool
		errContains string
	}{
		"empty notifier ID fails": {
			emailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:   "",
				MailingLists: []string{"user@example.com"},
			},
			setupMocks:  func(notifierDS *notifierDSMocks.MockDataStore) {},
			expectError: true,
			errContains: "must specify a valid email notifier",
		},
		"empty mailing list fails": {
			emailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:   "notifier-1",
				MailingLists: []string{},
			},
			setupMocks:  func(notifierDS *notifierDSMocks.MockDataStore) {},
			expectError: true,
			errContains: "must specify at least one email recipient",
		},
		"invalid email address fails": {
			emailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:   "notifier-1",
				MailingLists: []string{"invalid-email"},
			},
			setupMocks:  func(notifierDS *notifierDSMocks.MockDataStore) {},
			expectError: true,
			errContains: "invalid email recipient address",
		},
		"notifier not found fails": {
			emailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:   "missing-notifier",
				MailingLists: []string{"user@example.com"},
			},
			setupMocks: func(notifierDS *notifierDSMocks.MockDataStore) {
				notifierDS.EXPECT().Exists(gomock.Any(), "missing-notifier").Return(false, nil)
			},
			expectError: true,
			errContains: "not found",
		},
		"valid email config": {
			emailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:    "notifier-1",
				MailingLists:  []string{"user@example.com", "admin@example.com"},
				CustomSubject: "Test Report",
				CustomBody:    "Report body",
			},
			setupMocks: func(notifierDS *notifierDSMocks.MockDataStore) {
				notifierDS.EXPECT().Exists(gomock.Any(), "notifier-1").Return(true, nil)
			},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create fresh mocks for each subtest to avoid mock conflicts
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
			collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
			reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
			snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)

			validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

			tc.setupMocks(notifierDS)
			err := validator.validateEmailConfig(tc.emailConfig)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNodeResourceScope(t *testing.T) {
	t.Setenv(features.VulnerabilityReportsEnhancedFiltering.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
	collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
	reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
	snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)

	validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

	tests := map[string]struct {
		scope       *apiV2.ResourceScope
		expectError bool
		errContains string
	}{
		"collection scope not allowed for nodes": {
			scope: &apiV2.ResourceScope{
				ScopeReference: &apiV2.ResourceScope_CollectionScope{
					CollectionScope: &apiV2.CollectionReference{
						CollectionId: "collection-1",
					},
				},
			},
			expectError: true,
			errContains: "must use entity scope",
		},
		"nil entity scope fails": {
			scope: &apiV2.ResourceScope{
				ScopeReference: &apiV2.ResourceScope_EntityScope{
					EntityScope: nil,
				},
			},
			expectError: true,
			errContains: "cannot be nil",
		},
		"namespace scope not allowed for nodes": {
			scope: &apiV2.ResourceScope{
				ScopeReference: &apiV2.ResourceScope_EntityScope{
					EntityScope: &apiV2.EntityScope{
						Rules: []*apiV2.EntityScopeRule{
							{
								Entity: apiV2.ScopeEntity_SCOPE_ENTITY_NAMESPACE,
								Field:  apiV2.ScopeField_FIELD_NAME,
								Values: []*apiV2.RuleValue{
									{Value: "default", MatchType: apiV2.MatchType_EXACT},
								},
							},
						},
					},
				},
			},
			expectError: true,
			errContains: "only support cluster-level scoping",
		},
		"valid cluster scope": {
			scope: &apiV2.ResourceScope{
				ScopeReference: &apiV2.ResourceScope_EntityScope{
					EntityScope: &apiV2.EntityScope{
						Rules: []*apiV2.EntityScopeRule{
							{
								Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
								Field:  apiV2.ScopeField_FIELD_NAME,
								Values: []*apiV2.RuleValue{
									{Value: "prod", MatchType: apiV2.MatchType_EXACT},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validator.validateNodeResourceScope(tc.scope)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCollectionScope(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
	collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
	reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
	snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)

	validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

	tests := map[string]struct {
		collectionRef *apiV2.CollectionReference
		setupMocks    func()
		expectError   bool
		errContains   string
	}{
		"nil collection reference fails": {
			collectionRef: nil,
			setupMocks:    func() {},
			expectError:   true,
			errContains:   "must specify a valid collection ID",
		},
		"empty collection ID fails": {
			collectionRef: &apiV2.CollectionReference{
				CollectionId: "",
			},
			setupMocks:  func() {},
			expectError: true,
			errContains: "must specify a valid collection ID",
		},
		"collection not found fails": {
			collectionRef: &apiV2.CollectionReference{
				CollectionId: "missing-collection",
			},
			setupMocks: func() {
				collectionDS.EXPECT().Exists(gomock.Any(), "missing-collection").Return(false, nil)
			},
			expectError: true,
			errContains: "not found",
		},
		"valid collection": {
			collectionRef: &apiV2.CollectionReference{
				CollectionId: "collection-1",
			},
			setupMocks: func() {
				collectionDS.EXPECT().Exists(gomock.Any(), "collection-1").Return(true, nil)
			},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setupMocks()
			err := validator.validateCollectionScope(tc.collectionRef)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNodeFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
	collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
	reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
	snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)

	validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

	tests := map[string]struct {
		filters     *apiV2.NodeVulnerabilityReportFilters
		expectError bool
		errContains string
	}{
		"nil filters fail": {
			filters:     nil,
			expectError: true,
			errContains: "cannot be nil",
		},
		"missing CVE time filter fails": {
			filters: &apiV2.NodeVulnerabilityReportFilters{
				CvesSince: nil,
			},
			expectError: true,
			errContains: "must specify CVE time filter",
		},
		"valid all vulnerabilities filter": {
			filters: &apiV2.NodeVulnerabilityReportFilters{
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
			expectError: false,
		},
		"valid filter with query": {
			filters: &apiV2.NodeVulnerabilityReportFilters{
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
				Query: "Cluster:prod",
			},
			expectError: false,
		},
		"invalid query fails": {
			filters: &apiV2.NodeVulnerabilityReportFilters{
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
				Query: "invalid query syntax [",
			},
			expectError: true,
			errContains: "invalid query",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validator.validateNodeFilters(tc.filters)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateResourceScope(t *testing.T) {
	t.Setenv(features.VulnerabilityReportsEnhancedFiltering.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
	collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
	reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
	snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)

	validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

	tests := map[string]struct {
		config      *apiV2.ReportConfiguration
		setupMocks  func()
		expectError bool
		errContains string
	}{
		"nil scope fails": {
			config: &apiV2.ReportConfiguration{
				Type:          apiV2.ReportConfiguration_VULNERABILITY,
				ResourceScope: nil,
			},
			setupMocks:  func() {},
			expectError: true,
			errContains: "must specify a valid resource scope",
		},
		"valid collection scope for vulnerability report": {
			config: &apiV2.ReportConfiguration{
				Type: apiV2.ReportConfiguration_VULNERABILITY,
				ResourceScope: &apiV2.ResourceScope{
					ScopeReference: &apiV2.ResourceScope_CollectionScope{
						CollectionScope: &apiV2.CollectionReference{
							CollectionId: "collection-1",
						},
					},
				},
			},
			setupMocks: func() {
				collectionDS.EXPECT().Exists(gomock.Any(), "collection-1").Return(true, nil)
			},
			expectError: false,
		},
		"valid entity scope for vulnerability report": {
			config: &apiV2.ReportConfiguration{
				Type: apiV2.ReportConfiguration_VULNERABILITY,
				ResourceScope: &apiV2.ResourceScope{
					ScopeReference: &apiV2.ResourceScope_EntityScope{
						EntityScope: &apiV2.EntityScope{
							Rules: []*apiV2.EntityScopeRule{
								{
									Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
									Field:  apiV2.ScopeField_FIELD_NAME,
									Values: []*apiV2.RuleValue{
										{Value: "prod", MatchType: apiV2.MatchType_EXACT},
									},
								},
							},
						},
					},
				},
			},
			setupMocks:  func() {},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setupMocks()
			err := validator.validateResourceScope(tc.config)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCancelReportRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
	collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
	reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
	snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)

	validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

	tests := map[string]struct {
		reportID    string
		setupMocks  func()
		expectError bool
		errContains string
	}{
		"snapshot not found": {
			reportID: "missing-report",
			setupMocks: func() {
				snapshotDS.EXPECT().Get(gomock.Any(), "missing-report").Return(nil, false, nil)
			},
			expectError: true,
			errContains: "not found",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setupMocks()
			requester := &storage.SlimUser{
				Id:   "user-1",
				Name: "Test User",
			}
			err := validator.ValidateCancelReportRequest(tc.reportID, requester)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
