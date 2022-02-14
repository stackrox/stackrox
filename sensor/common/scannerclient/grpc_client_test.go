package scannerclient

import (
	"context"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

func TestGRPC(t *testing.T) {
	suite.Run(t, new(grpcSuite))
}

type grpcSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *grpcSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	testutils.LoadCustomTestMTLSCerts(s.envIsolator, path.Join(dir, "testdata"))
}

func (s *grpcSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *grpcSuite) TestClient() {
	client, err := newGRPCClient("scanner.another.svc:8443")
	s.Require().NoError(err)
	s.Require().NotNil(client)
	_, err = client.GetImageAnalysis(context.Background(), &storage.ContainerImage{
		Namespace: "stackrox",
		Name: &storage.ImageName{
			Tag: "latest",
			Registry: "docker.io/stackrox",
			Remote: "main",
			FullName: "docker.io/main:latest",
		},
	})
	s.Require().NoError(err)
}
