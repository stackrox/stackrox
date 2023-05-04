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

func (s *testSuite) TestGetMetadataManagedCentral() {
	srv := &serviceImpl{}
	managedCentralEnvVar := env.ManagedCentral.EnvVar()

	// ROX_MANAGED_CENTRAL not set
	actual, err := srv.GetMetadata(context.Background(), &v1.Empty{})
	s.NoError(err)
	s.Equal(env.ManagedCentral.DefaultBooleanSetting(), actual.IsManagedCentral)

	// ROX_MANAGED_CENTRAL set to false
	s.T().Setenv(managedCentralEnvVar, "false")
	actual, err = srv.GetMetadata(context.Background(), &v1.Empty{})
	s.NoError(err)
	s.Equal(false, actual.IsManagedCentral)

	// ROX_MANAGED_CENTRAL set to true
	s.T().Setenv(managedCentralEnvVar, "true")
	actual, err = srv.GetMetadata(context.Background(), &v1.Empty{})
	s.NoError(err)
	s.Equal(true, actual.IsManagedCentral)

}
