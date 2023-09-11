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
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type mockClustersServiceServer struct {
	v1.UnimplementedClustersServiceServer

	// injected behavior
	getKernelSupportInjectedFn getKernelSupportFn
	getDefaultsInjectedFn      getDefaultsFn
	postClusterInjectedFn      postClusterFn

	// spy properties
	clusterSent                     []storage.Cluster
	getClusterCalled                bool
	getKernelSupportAvailableCalled bool
}

type getKernelSupportFn func() (*v1.KernelSupportAvailableResponse, error)
type postClusterFn func(cluster *storage.Cluster) (*v1.ClusterResponse, error)
type getDefaultsFn func() (*v1.ClusterDefaultsResponse, error)

func (m *mockClustersServiceServer) GetClusterDefaultValues(_ context.Context, _ *v1.Empty) (*v1.ClusterDefaultsResponse, error) {
	return m.getDefaultsInjectedFn()
}

func (m *mockClustersServiceServer) GetKernelSupportAvailable(_ context.Context, _ *v1.Empty) (*v1.KernelSupportAvailableResponse, error) {
	m.getKernelSupportAvailableCalled = true
	return m.getKernelSupportInjectedFn()
}

func (m *mockClustersServiceServer) PostCluster(_ context.Context, cluster *storage.Cluster) (*v1.ClusterResponse, error) {
	m.clusterSent = append(m.clusterSent, *cluster)
	return m.postClusterInjectedFn(cluster)
}

func (m *mockClustersServiceServer) GetClusters(_ context.Context, _ *v1.GetClustersRequest) (*v1.ClustersList, error) {
	m.getClusterCalled = true
	return &v1.ClustersList{
		Clusters: []*storage.Cluster{
			{
				Name: "test-cluster",
				Id:   "cluster-id",
			},
		},
	}, nil
}

type sensorGenerateTestSuite struct {
	suite.Suite
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
func (s *sensorGenerateTestSuite) createGRPCMockClustersService(getDefaultsF getDefaultsFn, postClusterF postClusterFn) (*grpc.ClientConn, closeFunction, *mockClustersServiceServer) {
	// create an in-memory listener that does not require exposing any ports on the host
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	mock := mockClustersServiceServer{
		getDefaultsInjectedFn: getDefaultsF,
		postClusterInjectedFn: postClusterF,
	}
	v1.RegisterClustersServiceServer(server, &mock)

	// start the server
	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func (s *sensorGenerateTestSuite) createMockedCommand(getDefaultsF getDefaultsFn, postClusterF postClusterFn) (*bytes.Buffer, *bytes.Buffer, closeFunction, sensorGenerateCommand, *mockClustersServiceServer) {
	var out, errOut *bytes.Buffer
	conn, closeF, mock := s.createGRPCMockClustersService(getDefaultsF, postClusterF)
	cmd := sensorGenerateCommand{
		cluster: &storage.Cluster{},
	}
	cmd.env, out, errOut = s.newTestMockEnvironmentWithConn(conn)
	return out, errOut, closeF, cmd, mock
}

func (s *sensorGenerateTestSuite) SetupTest() {
	testutils.SetExampleVersion(s.T())
}

var emptyGetBundle = func(params apiparams.ClusterZip, _ string, _ time.Duration, _ environment.Environment) error {
	return nil
}

func legacyKernelSupport(flag bool) func() (*v1.KernelSupportAvailableResponse, error) {
	return func() (*v1.KernelSupportAvailableResponse, error) {
		return &v1.KernelSupportAvailableResponse{
			KernelSupportAvailable: flag,
		}, nil
	}
}

func getDefaultsUnimplemented() func() (*v1.ClusterDefaultsResponse, error) {
	return func() (*v1.ClusterDefaultsResponse, error) {
		return nil, status.Error(codes.Unimplemented, "GetClusterDefaultValues unimplemented")
	}
}

func getDefaultsFake(kernelSupport bool) func() (*v1.ClusterDefaultsResponse, error) {
	return func() (*v1.ClusterDefaultsResponse, error) {
		return &v1.ClusterDefaultsResponse{
			KernelSupportAvailable: kernelSupport,
		}, nil
	}
}

// postClusterFake base fake function for service.PostCluster that returns the same cluster with fake id
func postClusterFake(cluster *storage.Cluster) (*v1.ClusterResponse, error) {
	cluster.Id = "test-id"
	return &v1.ClusterResponse{
		Cluster: cluster,
	}, nil
}

// postClusterAlreadyExistsFake fake function for service.PostCluster that always returns error codes.AlreadyExists
func postClusterAlreadyExistsFake(_ *storage.Cluster) (*v1.ClusterResponse, error) {
	return nil, status.Error(codes.AlreadyExists, "Cluster Exists")
}

func (s *sensorGenerateTestSuite) TestHandleClusterAlreadyExists() {
	testCases := map[string]struct {
		// cluster setup
		continueIfExistsFlag bool
		clusterName          string
		postClusterF         postClusterFn

		// expectations
		expectError             bool
		expectGetClustersCalled bool
		expectBundleDownloaded  bool
	}{
		"Throw error if cluster exists": {
			continueIfExistsFlag:    false,
			postClusterF:            postClusterAlreadyExistsFake,
			clusterName:             "test-cluster",
			expectError:             true,
			expectGetClustersCalled: false,
			expectBundleDownloaded:  false,
		},
		"Should fetch bundle and download zip file if --continue-if-exists=true": {
			continueIfExistsFlag:    true,
			postClusterF:            postClusterAlreadyExistsFake,
			clusterName:             "test-cluster",
			expectError:             false,
			expectGetClustersCalled: true,
			expectBundleDownloaded:  true,
		},
		"Should get clusters and fail with error finding preexisting cluster": {
			continueIfExistsFlag:    true,
			postClusterF:            postClusterAlreadyExistsFake,
			clusterName:             "non-existing",
			expectError:             true,
			expectGetClustersCalled: true,
			expectBundleDownloaded:  false,
		},
		"If cluster doesn't exist, GetClusters API shouldn't be called": {
			continueIfExistsFlag:    true,
			postClusterF:            postClusterFake,
			clusterName:             "test-cluster",
			expectError:             false,
			expectGetClustersCalled: false,
			expectBundleDownloaded:  true,
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			_, _, closeF, generateCmd, mock := s.createMockedCommand(getDefaultsFake(true), testCase.postClusterF)
			defer closeF()

			// Setup generateCmd
			generateCmd.timeout = time.Duration(5) * time.Second
			generateCmd.continueIfExists = testCase.continueIfExistsFlag
			generateCmd.cluster.Name = testCase.clusterName
			getBundleCalled := false
			generateCmd.getBundleFn = func(_ apiparams.ClusterZip, _ string, _ time.Duration, _ environment.Environment) error {
				getBundleCalled = true
				return nil
			}

			// Create cluster
			err := generateCmd.fullClusterCreation()

			// Assertions
			if testCase.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			s.Assert().Equal(testCase.expectGetClustersCalled, mock.getClusterCalled)
			s.Assert().Equal(testCase.expectBundleDownloaded, getBundleCalled)
		})
	}
}

func (s *sensorGenerateTestSuite) TestMainImageDefaultAndOverride() {
	testCases := map[string]struct {
		getDefaultsF      getDefaultsFn
		mainImageOverride string

		// expected
		expectPostedClusterMainImage string
	}{
		"Without override: send empty main image": {
			getDefaultsF:                 getDefaultsFake(true),
			mainImageOverride:            "",
			expectPostedClusterMainImage: "",
		},
		"With override: send main image provided": {
			getDefaultsF:                 getDefaultsFake(true),
			mainImageOverride:            "some.registry.io/stackrox/main",
			expectPostedClusterMainImage: "some.registry.io/stackrox/main",
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			_, _, closeF, generateCmd, mock := s.createMockedCommand(testCase.getDefaultsF, postClusterFake)
			defer closeF()
			mock.getKernelSupportInjectedFn = legacyKernelSupport(true)

			// Setup generateCmd
			generateCmd.timeout = time.Duration(5) * time.Second
			generateCmd.getBundleFn = emptyGetBundle
			generateCmd.cluster.MainImage = testCase.mainImageOverride

			// Create cluster
			err := generateCmd.fullClusterCreation()
			s.Require().NoError(err, "shouldn't fail creating cluster")

			// Check that correct main image was posted
			s.Require().Len(mock.clusterSent, 1)
			s.Require().Equal(testCase.expectPostedClusterMainImage, mock.clusterSent[0].MainImage)
		})
	}
}

func (s *sensorGenerateTestSuite) TestLegacyAPICalledIfGetClustersUnimplemented() {
	_, _, closeF, generateCmd, mock := s.createMockedCommand(getDefaultsUnimplemented(), postClusterFake)
	defer closeF()
	mock.getKernelSupportInjectedFn = legacyKernelSupport(true)

	// Setup generateCmd
	generateCmd.timeout = time.Duration(5) * time.Second
	generateCmd.getBundleFn = emptyGetBundle

	// Create cluster
	err := generateCmd.fullClusterCreation()
	s.Require().NoError(err, "shouldn't fail creating cluster")

	// Check that legacy API was called
	s.Require().True(mock.getKernelSupportAvailableCalled)
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
		"No flags and kernel support in central: default to slim collector": {
			serverHasKernelSupport: true,
			slimCollectorFlag:      nil,
			warning:                nil,
			expectSlimMode:         true,
		},
		"No flags and no kernel support in central: default to full collector": {
			serverHasKernelSupport: false,
			slimCollectorFlag:      nil,
			warning:                nil,
			expectSlimMode:         false,
		},
		"--slim-collector=true and support in central: slim collector": {
			serverHasKernelSupport: true,
			slimCollectorFlag:      &slimFlag{true},
			warning:                nil,
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
			slimCollectorFlag:      &slimFlag{false},
			warning:                nil,
			expectSlimMode:         false,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		s.Run(name, func() {
			_, errOut, closeF, generateCmd, mock := s.createMockedCommand(getDefaultsFake(testCase.serverHasKernelSupport), postClusterFake)
			defer closeF()

			// Setup generateCmd
			if testCase.slimCollectorFlag != nil {
				generateCmd.slimCollectorP = &testCase.slimCollectorFlag.value
			}
			generateCmd.timeout = time.Duration(5) * time.Second
			var slimCollectorRequested *bool
			generateCmd.getBundleFn = func(params apiparams.ClusterZip, _ string, _ time.Duration, _ environment.Environment) error {
				slimCollectorRequested = params.SlimCollector
				return nil
			}

			// Create cluster
			err := generateCmd.fullClusterCreation()

			// Assertions
			s.Require().NoError(err)

			if testCase.warning != nil {
				s.Assert().Contains(errOut.String(), testCase.warning.messageTemplate)
			}

			s.Require().Len(mock.clusterSent, 1)
			s.Assert().Equal(mock.clusterSent[0].SlimCollector, testCase.expectSlimMode)
			s.Assert().Equal(*slimCollectorRequested, testCase.expectSlimMode)
		})
	}
}
