package validation

import (
	"testing"

	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	reportConfigDSMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	snapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	collectionDSMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	authnMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func emailNotifierConf(id string) *storage.NotifierConfiguration {
	return &storage.NotifierConfiguration{
		NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				NotifierId:   id,
				MailingLists: []string{"user@example.com"},
			},
		},
		Ref: &storage.NotifierConfiguration_Id{Id: id},
	}
}

func vulnReportConfigForRequestTests() *storage.ReportConfiguration {
	return &storage.ReportConfiguration{
		Id:   "vuln-report-config-1",
		Name: "Vulnerability Report",
		Type: storage.ReportConfiguration_VULNERABILITY,
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_CollectionId{CollectionId: "collection-1"},
		},
		Filter: &storage.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &storage.VulnerabilityReportFilters{},
		},
		Notifiers: []*storage.NotifierConfiguration{emailNotifierConf("notifier-1")},
		Creator:   &storage.SlimUser{Id: "creator-1", Name: "Config Creator"},
	}
}

func nodeReportConfigForRequestTests() *storage.ReportConfiguration {
	return &storage.ReportConfiguration{
		Id:   "node-report-config-1",
		Name: "Node CVE Report",
		Type: storage.ReportConfiguration_NODE_VULNERABILITY,
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_EntityScope{
				EntityScope: &storage.EntityScope{
					Rules: []*storage.EntityScopeRule{
						{
							Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
							Field:  storage.EntityField_FIELD_NAME,
							Values: []*storage.RuleValue{{Value: "cluster-a"}},
						},
					},
				},
			},
		},
		Filter: &storage.ReportConfiguration_NodeVulnReportFilters{
			NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{},
		},
		Notifiers: []*storage.NotifierConfiguration{emailNotifierConf("notifier-1")},
		Creator:   &storage.SlimUser{Id: "creator-1", Name: "Config Creator"},
	}
}

// requesterIdentity returns an identity mock for on-demand requests. Roles must
// be expected separately since not all request types consult them.
func requesterIdentity(ctrl *gomock.Controller) *authnMocks.MockIdentity {
	identity := authnMocks.NewMockIdentity(ctrl)
	identity.EXPECT().UID().Return("on-demand-user").AnyTimes()
	identity.EXPECT().FullName().Return("On Demand User").AnyTimes()
	identity.EXPECT().FriendlyName().Return("on-demand-user").AnyTimes()
	return identity
}

// TestValidateAndGenerateReportRequestOnDemandExtractsAccessScopeRules covers
// the ON_DEMAND branches of generateReportSnapshot for both report types: the
// generated snapshot must carry the access scope rules extracted from the
// requesting identity's roles via common.ExtractAccessScopeRules.
func TestValidateAndGenerateReportRequestOnDemandExtractsAccessScopeRules(t *testing.T) {
	customScope := &storage.SimpleAccessScope{
		Id: "custom-access-scope",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusterIds: []string{"cluster-a"},
		},
	}

	tests := map[string]struct {
		config *storage.ReportConfiguration
		// snapshotRules returns the access scope rules carried by the generated snapshot's report filters.
		snapshotRules func(snap *storage.ReportSnapshot) []*storage.SimpleAccessScope_Rules
		// configRules returns the access scope rules of the stored configuration's filters.
		configRules func(config *storage.ReportConfiguration) []*storage.SimpleAccessScope_Rules
	}{
		"vulnerability report": {
			config: vulnReportConfigForRequestTests(),
			snapshotRules: func(snap *storage.ReportSnapshot) []*storage.SimpleAccessScope_Rules {
				return snap.GetVulnReportFilters().GetAccessScopeRules()
			},
			configRules: func(config *storage.ReportConfiguration) []*storage.SimpleAccessScope_Rules {
				return config.GetVulnReportFilters().GetAccessScopeRules()
			},
		},
		"node vulnerability report": {
			config: nodeReportConfigForRequestTests(),
			snapshotRules: func(snap *storage.ReportSnapshot) []*storage.SimpleAccessScope_Rules {
				return snap.GetNodeVulnReportFilters().GetAccessScopeRules()
			},
			configRules: func(config *storage.ReportConfiguration) []*storage.SimpleAccessScope_Rules {
				return config.GetNodeVulnReportFilters().GetAccessScopeRules()
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
			snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)
			collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
			notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
			validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

			reportConfigDS.EXPECT().
				GetReportConfiguration(gomock.Any(), tc.config.GetId()).
				Return(tc.config, true, nil)
			if collectionID := tc.config.GetResourceScope().GetCollectionId(); collectionID != "" {
				collectionDS.EXPECT().
					Get(gomock.Any(), collectionID).
					Return(&storage.ResourceCollection{Id: collectionID, Name: "Collection One"}, true, nil)
			}
			notifierDS.EXPECT().
				GetManyNotifiers(gomock.Any(), []string{"notifier-1"}).
				Return([]*storage.Notifier{{Id: "notifier-1", Name: "Email Notifier"}}, nil)

			role := permissionsMocks.NewMockResolvedRole(ctrl)
			role.EXPECT().GetAccessScope().Return(customScope).Times(1)
			identity := requesterIdentity(ctrl)
			identity.EXPECT().Roles().Return([]permissions.ResolvedRole{role}).Times(1)

			reportReq, err := validator.ValidateAndGenerateReportRequest(
				tc.config.GetId(), storage.ReportStatus_EMAIL, storage.ReportStatus_ON_DEMAND, identity)
			require.NoError(t, err)

			snapshot := reportReq.ReportSnapshot
			require.NotNil(t, snapshot)

			// On-demand reports are attributed to the requesting identity...
			assert.Equal(t, "on-demand-user", snapshot.GetRequester().GetId())
			assert.Equal(t, "On Demand User", snapshot.GetRequester().GetName())
			assert.Equal(t, storage.ReportStatus_ON_DEMAND, snapshot.GetReportStatus().GetReportRequestType())
			assert.Equal(t, storage.ReportStatus_WAITING, snapshot.GetReportStatus().GetRunState())
			// ...and carry the access scope rules extracted from that identity's roles.
			assert.Equal(t, []*storage.SimpleAccessScope_Rules{customScope.GetRules()}, tc.snapshotRules(snapshot))
			// The snapshot works on a clone: the stored configuration's filters must stay untouched.
			assert.Empty(t, tc.configRules(tc.config))
		})
	}
}

// TestValidateAndGenerateReportRequestScheduledSkipsAccessScopeExtraction covers
// the SCHEDULED branches of generateReportSnapshot: the requester falls back to
// the report configuration creator and the identity's roles are never consulted,
// so no access scope rules are extracted.
func TestValidateAndGenerateReportRequestScheduledSkipsAccessScopeExtraction(t *testing.T) {
	tests := map[string]struct {
		config *storage.ReportConfiguration
		// snapshotRules returns the access scope rules carried by the generated snapshot's report filters.
		snapshotRules func(snap *storage.ReportSnapshot) []*storage.SimpleAccessScope_Rules
	}{
		"vulnerability report": {
			config: vulnReportConfigForRequestTests(),
			snapshotRules: func(snap *storage.ReportSnapshot) []*storage.SimpleAccessScope_Rules {
				return snap.GetVulnReportFilters().GetAccessScopeRules()
			},
		},
		"node vulnerability report": {
			config: nodeReportConfigForRequestTests(),
			snapshotRules: func(snap *storage.ReportSnapshot) []*storage.SimpleAccessScope_Rules {
				return snap.GetNodeVulnReportFilters().GetAccessScopeRules()
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			reportConfigDS := reportConfigDSMocks.NewMockDataStore(ctrl)
			snapshotDS := snapshotDSMocks.NewMockDataStore(ctrl)
			collectionDS := collectionDSMocks.NewMockDataStore(ctrl)
			notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
			validator := New(reportConfigDS, snapshotDS, collectionDS, notifierDS)

			reportConfigDS.EXPECT().
				GetReportConfiguration(gomock.Any(), tc.config.GetId()).
				Return(tc.config, true, nil)
			if collectionID := tc.config.GetResourceScope().GetCollectionId(); collectionID != "" {
				collectionDS.EXPECT().
					Get(gomock.Any(), collectionID).
					Return(&storage.ResourceCollection{Id: collectionID, Name: "Collection One"}, true, nil)
			}
			notifierDS.EXPECT().
				GetManyNotifiers(gomock.Any(), []string{"notifier-1"}).
				Return([]*storage.Notifier{{Id: "notifier-1", Name: "Email Notifier"}}, nil)

			// No expectations are set on the identity: scheduled reports must not
			// consult the requester's roles, so any call fails the test.
			identity := authnMocks.NewMockIdentity(ctrl)

			reportReq, err := validator.ValidateAndGenerateReportRequest(
				tc.config.GetId(), storage.ReportStatus_EMAIL, storage.ReportStatus_SCHEDULED, identity)
			require.NoError(t, err)

			snapshot := reportReq.ReportSnapshot
			require.NotNil(t, snapshot)

			assert.Equal(t, storage.ReportStatus_SCHEDULED, snapshot.GetReportStatus().GetReportRequestType())
			// Scheduled reports are attributed to the configuration creator...
			assert.Equal(t, tc.config.GetCreator(), snapshot.GetRequester())
			// ...and carry no access scope rules.
			assert.Empty(t, tc.snapshotRules(snapshot))
		})
	}
}

// TestGenerateReportSnapshotAccessScopeRules exercises generateReportSnapshot
// directly, covering the identity shapes that ExtractAccessScopeRules resolves
// at its call sites: custom scopes, the unrestricted scope, and the exclude-all
// fallbacks, plus the request types and nil-filter configurations that skip the
// extraction entirely.
func TestGenerateReportSnapshotAccessScopeRules(t *testing.T) {
	customScope := &storage.SimpleAccessScope{
		Id: "custom-access-scope",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusterIds: []string{"cluster-a"},
		},
	}
	excludeAllRules := rolePkg.AccessScopeExcludeAll.GetRules()

	vulnConfig := func(filters *storage.VulnerabilityReportFilters) *storage.ReportConfiguration {
		return &storage.ReportConfiguration{
			Id:            "vuln-config",
			Name:          "Vulnerability Report",
			Type:          storage.ReportConfiguration_VULNERABILITY,
			ResourceScope: &storage.ResourceScope{ScopeReference: &storage.ResourceScope_CollectionId{CollectionId: "collection-1"}},
			Filter:        &storage.ReportConfiguration_VulnReportFilters{VulnReportFilters: filters},
		}
	}
	nodeConfig := func(filters *storage.NodeVulnerabilityReportFilters) *storage.ReportConfiguration {
		return &storage.ReportConfiguration{
			Id:   "node-config",
			Name: "Node CVE Report",
			Type: storage.ReportConfiguration_NODE_VULNERABILITY,
			ResourceScope: &storage.ResourceScope{
				ScopeReference: &storage.ResourceScope_EntityScope{
					EntityScope: &storage.EntityScope{
						Rules: []*storage.EntityScopeRule{
							{
								Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
								Field:  storage.EntityField_FIELD_NAME,
								Values: []*storage.RuleValue{{Value: "cluster-a"}},
							},
						},
					},
				},
			},
			Filter: &storage.ReportConfiguration_NodeVulnReportFilters{NodeVulnReportFilters: filters},
		}
	}

	tests := map[string]struct {
		config      *storage.ReportConfiguration
		requestType storage.ReportStatus_RunMethod
		// consultsRoles is true when the snapshot generation is expected to read
		// the identity's roles exactly once (ON_DEMAND with non-nil filters).
		consultsRoles bool
		// makeRoles builds the roles reported by the identity.
		makeRoles func(ctrl *gomock.Controller) []permissions.ResolvedRole
		verify    func(t *testing.T, snapshot *storage.ReportSnapshot)
	}{
		"on demand vulnerability report extracts access scope rules from the requester's roles": {
			config:        vulnConfig(&storage.VulnerabilityReportFilters{}),
			requestType:   storage.ReportStatus_ON_DEMAND,
			consultsRoles: true,
			makeRoles: func(ctrl *gomock.Controller) []permissions.ResolvedRole {
				role := permissionsMocks.NewMockResolvedRole(ctrl)
				role.EXPECT().GetAccessScope().Return(customScope).Times(1)
				return []permissions.ResolvedRole{role}
			},
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				assert.Equal(t, []*storage.SimpleAccessScope_Rules{customScope.GetRules()},
					snapshot.GetVulnReportFilters().GetAccessScopeRules())
			},
		},
		"on demand node vulnerability report extracts access scope rules from the requester's roles": {
			config:        nodeConfig(&storage.NodeVulnerabilityReportFilters{}),
			requestType:   storage.ReportStatus_ON_DEMAND,
			consultsRoles: true,
			makeRoles: func(ctrl *gomock.Controller) []permissions.ResolvedRole {
				role := permissionsMocks.NewMockResolvedRole(ctrl)
				role.EXPECT().GetAccessScope().Return(customScope).Times(1)
				return []permissions.ResolvedRole{role}
			},
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				assert.Equal(t, []*storage.SimpleAccessScope_Rules{customScope.GetRules()},
					snapshot.GetNodeVulnReportFilters().GetAccessScopeRules())
			},
		},
		"on demand report for an unrestricted role extracts no rules": {
			config:        vulnConfig(&storage.VulnerabilityReportFilters{}),
			requestType:   storage.ReportStatus_ON_DEMAND,
			consultsRoles: true,
			makeRoles: func(ctrl *gomock.Controller) []permissions.ResolvedRole {
				role := permissionsMocks.NewMockResolvedRole(ctrl)
				role.EXPECT().GetAccessScope().Return(rolePkg.AccessScopeIncludeAll).Times(1)
				return []permissions.ResolvedRole{role}
			},
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				// Nil rules mean unrestricted access to all clusters/namespaces.
				assert.Nil(t, snapshot.GetVulnReportFilters().GetAccessScopeRules())
			},
		},
		"on demand report for a roleless identity falls back to the exclude-all scope": {
			config:        vulnConfig(&storage.VulnerabilityReportFilters{}),
			requestType:   storage.ReportStatus_ON_DEMAND,
			consultsRoles: true,
			makeRoles: func(_ *gomock.Controller) []permissions.ResolvedRole {
				return nil
			},
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				assert.Equal(t, []*storage.SimpleAccessScope_Rules{excludeAllRules},
					snapshot.GetVulnReportFilters().GetAccessScopeRules())
			},
		},
		"on demand report for a role without an access scope falls back to the exclude-all scope": {
			config:        vulnConfig(&storage.VulnerabilityReportFilters{}),
			requestType:   storage.ReportStatus_ON_DEMAND,
			consultsRoles: true,
			makeRoles: func(ctrl *gomock.Controller) []permissions.ResolvedRole {
				role := permissionsMocks.NewMockResolvedRole(ctrl)
				role.EXPECT().GetAccessScope().Return(nil).Times(1)
				return []permissions.ResolvedRole{role}
			},
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				assert.Equal(t, []*storage.SimpleAccessScope_Rules{excludeAllRules},
					snapshot.GetVulnReportFilters().GetAccessScopeRules())
			},
		},
		"scheduled vulnerability report does not extract access scope rules": {
			config:      vulnConfig(&storage.VulnerabilityReportFilters{}),
			requestType: storage.ReportStatus_SCHEDULED,
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				assert.Empty(t, snapshot.GetVulnReportFilters().GetAccessScopeRules())
			},
		},
		"scheduled node vulnerability report does not extract access scope rules": {
			config:      nodeConfig(&storage.NodeVulnerabilityReportFilters{}),
			requestType: storage.ReportStatus_SCHEDULED,
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				assert.Empty(t, snapshot.GetNodeVulnReportFilters().GetAccessScopeRules())
			},
		},
		"on demand vulnerability report without filters skips the extraction": {
			config:        vulnConfig(nil),
			requestType:   storage.ReportStatus_ON_DEMAND,
			consultsRoles: false,
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				assert.Nil(t, snapshot.GetFilter())
			},
		},
		"on demand node vulnerability report without filters skips the extraction": {
			config:        nodeConfig(nil),
			requestType:   storage.ReportStatus_ON_DEMAND,
			consultsRoles: false,
			verify: func(t *testing.T, snapshot *storage.ReportSnapshot) {
				assert.Nil(t, snapshot.GetFilter())
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			identity := authnMocks.NewMockIdentity(ctrl)
			if tc.requestType == storage.ReportStatus_ON_DEMAND {
				// ON_DEMAND snapshots are attributed to the requesting identity.
				identity.EXPECT().UID().Return("on-demand-user").AnyTimes()
				identity.EXPECT().FullName().Return("On Demand User").AnyTimes()
				identity.EXPECT().FriendlyName().Return("on-demand-user").AnyTimes()
			}
			if tc.consultsRoles {
				var roles []permissions.ResolvedRole
				if tc.makeRoles != nil {
					roles = tc.makeRoles(ctrl)
				}
				identity.EXPECT().Roles().Return(roles).Times(1)
			}
			// For the remaining cases no Roles expectation is set: any call
			// would fail the test, proving the extraction is skipped.

			snapshot := generateReportSnapshot(tc.config, nil, nil, storage.ReportStatus_DOWNLOAD, tc.requestType, identity)
			require.NotNil(t, snapshot)

			if tc.requestType == storage.ReportStatus_ON_DEMAND {
				assert.Equal(t, "on-demand-user", snapshot.GetRequester().GetId())
				assert.Equal(t, "On Demand User", snapshot.GetRequester().GetName())
			} else {
				// Scheduled snapshots are attributed to the configuration creator.
				assert.Equal(t, tc.config.GetCreator(), snapshot.GetRequester())
			}
			assert.Equal(t, tc.requestType, snapshot.GetReportStatus().GetReportRequestType())

			tc.verify(t, snapshot)
		})
	}
}
