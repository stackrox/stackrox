package generate

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type mockClustersServiceServer struct {
	v1.ClustersServiceServer

	actAsLegacyCentral bool
	kernelSupport      bool
	clusterSent        []storage.Cluster
}

func (m *mockClustersServiceServer) GetKernelSupportAvailable(ctx context.Context, in *v1.Empty) (*v1.KernelSupportAvailableResponse, error) {
	return &v1.KernelSupportAvailableResponse{
		KernelSupportAvailable: m.kernelSupport,
	}, nil
}

func (m *mockClustersServiceServer) PostCluster(ctx context.Context, cluster *storage.Cluster) (*v1.ClusterResponse, error) {
	m.clusterSent = append(m.clusterSent, *cluster)
	if m.actAsLegacyCentral && cluster.MainImage == "" {
		return nil, status.Error(codes.Internal, "Cluster Validation error: invalid main image '': invalid reference format")
	}

	cluster.Id = "test-id"
	return &v1.ClusterResponse{
		Cluster: cluster,
	}, nil
}

type sensorGenerateTestSuite struct {
	suite.Suite
	cmd sensorGenerateCommand
}

type expectedWarning struct {
	messageTemplate string
}

func TestSensorGenerateCommand(t *testing.T) {
	suite.Run(t, new(sensorGenerateTestSuite))
}

type closeFunction = func()

// createGRPCMockClustersService will create an in-memory gRPC server serving mockClustersServiceServer
// NOTE: Ensure that you ALWAYS call the closeFunction to clean up the test setup
func (s *sensorGenerateTestSuite) createGRPCMockClustersService(kernelSupport bool, legacyCentral bool) (*grpc.ClientConn, closeFunction, *mockClustersServiceServer) {
	// create an in-memory listener that does not require exposing any ports on the host
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	mock := mockClustersServiceServer{kernelSupport: kernelSupport, actAsLegacyCentral: legacyCentral}
	v1.RegisterClustersServiceServer(server, &mock)

	// start the server
	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithInsecure())
	s.Require().NoError(err)

	closeF := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, closeF, &mock
}

func (s *sensorGenerateTestSuite) newTestMockEnvironmentWithConn(conn *grpc.ClientConn) (environment.Environment, *bytes.Buffer, *bytes.Buffer) {
	return mocks.NewEnvWithConn(conn, s.T())
}

func (s *sensorGenerateTestSuite) createMockedCommand(kernelSupport bool, legacyCentral bool) (*bytes.Buffer, *bytes.Buffer, closeFunction, sensorGenerateCommand, *mockClustersServiceServer) {
	var out, errOut *bytes.Buffer
	conn, closeF, mock := s.createGRPCMockClustersService(kernelSupport, legacyCentral)
	cmd := s.cmd
	cmd.env, out, errOut = s.newTestMockEnvironmentWithConn(conn)
	return out, errOut, closeF, cmd, mock
}

// Create cluster:
//   2) Return validation errors
//   3) Return "AlreadyExists" error: if existing cluster, try to fetch cluster
//   4) Correctly create cluster
// Get bundle:
//	 1) Show warnings / info messages
//   2) Return error
// Flag combinations:
//   1) Show different warnings or errors depending on the flag combinations

func (s *sensorGenerateTestSuite) SetupTest() {
	testbuildinfo.SetForTest(s.T())
	testutils.SetExampleVersion(s.T())
}

var emptyGetBundle = func(params apiparams.ClusterZip, _ string, _ time.Duration) error {
	return nil
}

// getMainImageFromBuildFlag is necessary because we run tests both in release and non-release mode.
// roxctl will select different images based on the build type if communicating with a legacy central
func getMainImageFromBuildFlag() string {
	var flavor defaults.ImageFlavor
	if buildinfo.ReleaseBuild {
		flavor = defaults.RHACSReleaseImageFlavor()
	} else {
		flavor = defaults.DevelopmentBuildImageFlavor()
	}
	return flavor.MainImageNoTag()
}

func (s *sensorGenerateTestSuite) TestResendClusterIfLegacyCentral() {
	testCases := map[string]struct {
		legacyCentral bool

		// expected
		expectedClustersSent       int
		expectedClustersMainImages []string
		expectedWarning            *expectedWarning
	}{
		"legacy central: PostCluster called twice": {
			legacyCentral:              true,
			expectedClustersSent:       2,
			expectedClustersMainImages: []string{"", getMainImageFromBuildFlag()},
			expectedWarning:            &expectedWarning{"Running older version of central"},
		},
		"new central: PostCluster called once without MainImage": {
			legacyCentral:              false,
			expectedClustersSent:       1,
			expectedClustersMainImages: []string{""},
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			_, errOut, closeF, generateCmd, mock := s.createMockedCommand(true, testCase.legacyCentral)
			defer closeF()
			generateCmd.timeout = time.Duration(5) * time.Second
			generateCmd.getBundleFn = emptyGetBundle
			err := generateCmd.fullClusterCreation()
			s.Require().NoError(err)

			if testCase.expectedWarning != nil {
				s.Assert().Contains(errOut.String(), testCase.expectedWarning.messageTemplate)
			}

			s.Assert().Len(mock.clusterSent, testCase.expectedClustersSent)
			for i, mainImage := range testCase.expectedClustersMainImages {
				s.Assert().Equal(mock.clusterSent[i].MainImage, mainImage)
			}
		})
	}

}

func (s *sensorGenerateTestSuite) TestSlimCollectorSelection() {
	type slimFlag struct {
		value bool
	}

	var testCases = map[string]struct {
		serverHasKernelSupport bool
		slimCollectorFlag      *slimFlag

		// expectations
		warning        *expectedWarning
		expectSlimMode bool
	}{
		"no flags and kernel support in central: default to slim collector": {
			serverHasKernelSupport: true,
			expectSlimMode:         true,
		},
		"no flags and no kernel support in central: default to full collector": {
			serverHasKernelSupport: false,
			expectSlimMode:         false,
		},
		"--slim-collector=true and support in central: slim collector": {
			serverHasKernelSupport: true,
			slimCollectorFlag:      &slimFlag{true},
			expectSlimMode:         true,
		},
		"--slim-collector=true and no kernel support in central: slim collector + warning": {
			serverHasKernelSupport: false,
			slimCollectorFlag:      &slimFlag{true},
			warning:                &expectedWarning{"The deployment bundle will reference a slim collector image"},
			expectSlimMode:         true,
		},
		"--slim-collector=false: collector full": {
			serverHasKernelSupport: true,
			slimCollectorFlag:      &slimFlag{true},
			expectSlimMode:         true,
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			_, errOut, closeF, generateCmd, mock := s.createMockedCommand(testCase.serverHasKernelSupport, false)
			defer closeF()
			if testCase.slimCollectorFlag != nil {
				generateCmd.slimCollectorP = &testCase.slimCollectorFlag.value
			}
			generateCmd.timeout = time.Duration(5) * time.Second
			var slimCollectorRequested *bool
			generateCmd.getBundleFn = func(params apiparams.ClusterZip, _ string, _ time.Duration) error {
				slimCollectorRequested = params.SlimCollector
				return nil
			}

			err := generateCmd.fullClusterCreation()
			s.Require().NoError(err)

			if testCase.warning != nil {
				s.Assert().Contains(errOut.String(), testCase.warning.messageTemplate)
			}

			s.Assert().Len(mock.clusterSent, 1)
			s.Assert().Equal(mock.clusterSent[0].SlimCollector, testCase.expectSlimMode)
			s.Assert().Equal(*slimCollectorRequested, testCase.expectSlimMode)
		})
	}
}
