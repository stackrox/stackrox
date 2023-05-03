package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/suite"
)

type testSuite struct {
	suite.Suite
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) TestMetadataManagedCentral() {
	srv := &serviceImpl{}

	// ROX_MANAGED_CENTRAL not set
	actual, err := srv.GetMetadata(context.Background(), &v1.Empty{})
	s.NoError(err)
	env.ManagedCentral.BooleanSetting()
	s.EqualValues(false, actual.IsManagedCentral)

	// ROX_MANAGED_CENTRAL set to false
	s.T().Setenv("ROX_MANAGED_CENTRAL", "false")
	actual, err = srv.GetMetadata(context.Background(), &v1.Empty{})
	s.NoError(err)
	s.EqualValues(false, actual.IsManagedCentral)

	// ROX_MANAGED_CENTRAL set to true
	s.T().Setenv("ROX_MANAGED_CENTRAL", "true")
	actual, err = srv.GetMetadata(context.Background(), &v1.Empty{})
	s.NoError(err)
	s.EqualValues(true, actual.IsManagedCentral)

}
