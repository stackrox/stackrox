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

	// injected behavior
	getKernelSupportInjectedFn getKernelSupportFn
	getDefaultsInjectedFn getDefaultsFn
	postClusterInjectedFn postClusterFn

	// spy properties
	clusterSent      []storage.Cluster
	getClusterCalled bool
}

type getKernelSupportFn func() (*v1.KernelSupportAvailableResponse, error)
type postClusterFn func(cluster *storage.Cluster) (*v1.ClusterResponse, error)
type getDefaultsFn func() (*v1.ClusterDefaultsResponse, error)

func (m *mockClustersServiceServer) GetClusterDefaults(ctx context.Context, in *v1.Empty) (*v1.ClusterDefaultsResponse, error) {
	return m.getDefaultsInjectedFn()
}

func (m *mockClustersServiceServer) GetKernelSupportAvailable(ctx context.Context, in *v1.Empty) (*v1.KernelSupportAvailableResponse, error) {
	return m.getKernelSupportInjectedFn()
}

func (m *mockClustersServiceServer) PostCluster(ctx context.Context, cluster *storage.Cluster) (*v1.ClusterResponse, error) {
	m.clusterSent = append(m.clusterSent, *cluster)
	return m.postClusterInjectedFn(cluster)
}

func (m *mockClustersServiceServer) GetClusters(ctx context.Context, in *v1.GetClustersRequest) (*v1.ClustersList, error) {
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

type mockDefaults struct {
	kernelSupport bool
	mainImage     string
}

func (s *sensorGenerateTestSuite) createMockedCommand(getDefaultsF getDefaultsFn, postClusterF postClusterFn) (*bytes.Buffer, *bytes.Buffer, closeFunction, sensorGenerateCommand, *mockClustersServiceServer) {
	var out, errOut *bytes.Buffer
	conn, closeF, mock := s.createGRPCMockClustersService(getDefaultsF, postClusterF)
	cmd := s.cmd
	cmd.env, out, errOut = s.newTestMockEnvironmentWithConn(conn)
	return out, errOut, closeF, cmd, mock
}

func (s *sensorGenerateTestSuite) SetupTest() {
	testbuildinfo.SetForTest(s.T())
	testutils.SetExampleVersion(s.T())
}

var emptyGetBundle = func(params apiparams.ClusterZip, _ string, _ time.Duration) error {
	return nil
}

func legacyKernelSupport(flag bool) func() (*v1.KernelSupportAvailableResponse, error) {
	return func() (*v1.KernelSupportAvailableResponse, error) {
		return &v1.KernelSupportAvailableResponse{
			KernelSupportAvailable: flag,
		}, nil
	}
}

func defaultsUnimplemented() func() (*v1.ClusterDefaultsResponse, error) {
	return func() (*v1.ClusterDefaultsResponse, error) {
		return nil, status.Error(codes.Unimplemented, "GetClusterDefaults unimplemented")
	}
}

func defaultsFake(kernelSupport bool, mainImage string) func() (*v1.ClusterDefaultsResponse, error) {
	return func() (*v1.ClusterDefaultsResponse, error) {
		return &v1.ClusterDefaultsResponse{
			MainImageRepository:    mainImage,
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

// postClusterLegacyCentralFake fake legacy central function for service.PostCluster that returns validation error if main is empty
func postClusterLegacyCentralFake(cluster *storage.Cluster) (*v1.ClusterResponse, error) {
	if cluster.MainImage == "" {
		return nil, status.Error(codes.Internal, "Cluster Validation error: invalid main image '': invalid reference format")
	}
	return postClusterFake(cluster)
}

// postClusterAlreadyExistsFake fake function for service.PostCluster that always returns error codes.AlreadyExists
func postClusterAlreadyExistsFake(cluster *storage.Cluster) (*v1.ClusterResponse, error) {
	return nil, status.Error(codes.AlreadyExists, "Cluster Exists")
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
			_, _, closeF, generateCmd, mock := s.createMockedCommand(defaultsFake(true, "example.io/main"), testCase.postClusterF)
			defer closeF()

			// Setup generateCmd
			generateCmd.timeout = time.Duration(5) * time.Second
			generateCmd.continueIfExists = testCase.continueIfExistsFlag
			generateCmd.cluster.Name = testCase.clusterName
			getBundleCalled := false
			generateCmd.getBundleFn = func(_ apiparams.ClusterZip, _ string, _ time.Duration) error {
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
	flavorInTest := getFlavorFromReleaseBuild()
	testCases := map[string]struct {
		getDefaultsF      getDefaultsFn
		mainImageOverride string

		// expected
		expectPostedClusterMainImage string
	}{
		"New Central without override: use main image default": {
			getDefaultsF:                 defaultsFake(true, "example.io/main"),
			mainImageOverride:            "",
			expectPostedClusterMainImage: "example.io/main",
		},
		"New Central with override: use main image provided": {
			getDefaultsF:                 defaultsFake(true, "example.io/main"),
			mainImageOverride:            "some.registry.io/stackrox/main",
			expectPostedClusterMainImage: "some.registry.io/stackrox/main",
		},
		"Legacy Central without override: derive main image from release flag": {
			getDefaultsF:                 defaultsUnimplemented(),
			mainImageOverride:            "",
			expectPostedClusterMainImage: flavorInTest.MainImageNoTag(),
		},
		"Legacy Central with override: use main image provided": {
			getDefaultsF:                 defaultsUnimplemented(),
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
		s.Run(name, func() {
			_, errOut, closeF, generateCmd, mock := s.createMockedCommand(defaultsFake(testCase.serverHasKernelSupport, "example.io/main"), postClusterFake)
			defer closeF()

			// Setup generateCmd
			if testCase.slimCollectorFlag != nil {
				generateCmd.slimCollectorP = &testCase.slimCollectorFlag.value
			}
			generateCmd.timeout = time.Duration(5) * time.Second
			var slimCollectorRequested *bool
			generateCmd.getBundleFn = func(params apiparams.ClusterZip, _ string, _ time.Duration) error {
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
