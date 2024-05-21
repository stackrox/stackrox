package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	resultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSomeTest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	scanConfigDS := scanConfigMocks.NewMockDataStore(mockCtrl)
	resultDatastore := resultMocks.NewMockDataStore(mockCtrl)

	resultDatastore.EXPECT().ComplianceProfileResultStats(gomock.Any(), gomock.Any()).Return(
		[]*datastore.ResourceResultCountByProfile{
			{ProfileName: "cis-ocp", ErrorCount: 3, PassCount: 33},
			{ProfileName: "nist", ErrorCount: 5, PassCount: 55},
		}, nil,
	)

	service := &serviceImpl{
		scanConfigDS:        scanConfigDS,
		complianceResultsDS: resultDatastore,
	}
	ctx := sac.WithAllAccess(context.TODO())
	resp, err := service.GetComplianceProfilesStats(ctx, &v2.RawQuery{})
	require.NotNil(t, resp)
	require.NoError(t, err)
}
