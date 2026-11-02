package client

import (
	"context"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/scannerv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testImageDigest = "registry.example.com/repo@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f"

// fakeIndexerClient implements v4.IndexerClient, delegating only
// CreateIndexReport to the configured func; any other method panics via the
// embedded nil interface, which is fine since the tests here don't call them.
type fakeIndexerClient struct {
	v4.IndexerClient
	createIndexReportFunc func(ctx context.Context, in *v4.CreateIndexReportRequest, opts ...grpc.CallOption) (*v4.IndexReport, error)
}

func (f *fakeIndexerClient) CreateIndexReport(ctx context.Context, in *v4.CreateIndexReportRequest, opts ...grpc.CallOption) (*v4.IndexReport, error) {
	return f.createIndexReportFunc(ctx, in, opts...)
}

// fakeMatcherClient implements v4.MatcherClient, delegating only
// GetVulnerabilities to the configured func.
type fakeMatcherClient struct {
	v4.MatcherClient
	getVulnerabilitiesFunc func(ctx context.Context, in *v4.GetVulnerabilitiesRequest, opts ...grpc.CallOption) (*v4.VulnerabilityReport, error)
}

func (f *fakeMatcherClient) GetVulnerabilities(ctx context.Context, in *v4.GetVulnerabilitiesRequest, opts ...grpc.CallOption) (*v4.VulnerabilityReport, error) {
	return f.getVulnerabilitiesFunc(ctx, in, opts...)
}

func TestSetServiceVersion(t *testing.T) {
	tests := map[string]struct {
		version          *scannerv4.Version
		responseMetadata metadata.MD
		expectedVersion  string
	}{
		"version from metadata is used": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, "v4.8.0"),
			expectedVersion:  "v4.8.0",
		},
		"nil metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: nil,
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"empty metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.MD{},
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"header missing from metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs("other-header", "value"),
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"empty string version in metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, ""),
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"whitespace-only version in metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, "   "),
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"tab whitespace version in metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, "\t\n"),
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"nil version is a no-op": {
			version:          nil,
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, "v4.8.0"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			options := callOptions{version: tt.version}
			setMatcherVersion(options, tt.responseMetadata)
			setIndexerVersion(options, tt.responseMetadata)

			if tt.version == nil {
				assert.Nil(t, options.version)
			} else {
				assert.Equal(t, tt.expectedVersion, tt.version.Matcher, "matcher version")
				assert.Equal(t, tt.expectedVersion, tt.version.Indexer, "indexer version")
			}
		})
	}
}

func TestCreateAndScanImage(t *testing.T) {
	ref, err := name.NewDigest(testImageDigest)
	require.NoError(t, err)

	t.Run("indexer not configured", func(t *testing.T) {
		c := &gRPCScanner{matcher: &fakeMatcherClient{}}
		_, err := c.CreateAndScanImage(context.Background(), ref, authn.Anonymous, ImageRegistryOpt{})
		assert.ErrorIs(t, err, errIndexerNotConfigured)
	})

	t.Run("matcher not configured", func(t *testing.T) {
		c := &gRPCScanner{indexer: &fakeIndexerClient{}}
		_, err := c.CreateAndScanImage(context.Background(), ref, authn.Anonymous, ImageRegistryOpt{})
		assert.ErrorIs(t, err, errMatcherNotConfigured)
	})

	t.Run("CreateIndexReport failure is wrapped and not retried", func(t *testing.T) {
		calls := 0
		c := &gRPCScanner{
			indexer: &fakeIndexerClient{
				createIndexReportFunc: func(_ context.Context, _ *v4.CreateIndexReportRequest, _ ...grpc.CallOption) (*v4.IndexReport, error) {
					calls++
					// A non-retryable status code so the test doesn't wait out the backoff.
					return nil, status.Error(codes.InvalidArgument, "bad request")
				},
			},
			matcher: &fakeMatcherClient{},
		}
		_, err := c.CreateAndScanImage(context.Background(), ref, authn.Anonymous, ImageRegistryOpt{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create index")
		assert.Equal(t, 1, calls)
	})

	t.Run("always creates a fresh index report and returns the matcher's vulnerability report", func(t *testing.T) {
		var capturedCreateReq *v4.CreateIndexReportRequest
		var capturedVulnReq *v4.GetVulnerabilitiesRequest
		wantReport := &v4.VulnerabilityReport{HashId: "/v4/containerimage/sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f"}

		c := &gRPCScanner{
			indexer: &fakeIndexerClient{
				createIndexReportFunc: func(_ context.Context, in *v4.CreateIndexReportRequest, _ ...grpc.CallOption) (*v4.IndexReport, error) {
					capturedCreateReq = in
					return &v4.IndexReport{HashId: in.GetHashId(), Success: true}, nil
				},
			},
			matcher: &fakeMatcherClient{
				getVulnerabilitiesFunc: func(_ context.Context, in *v4.GetVulnerabilitiesRequest, _ ...grpc.CallOption) (*v4.VulnerabilityReport, error) {
					capturedVulnReq = in
					return wantReport, nil
				},
			},
		}

		got, err := c.CreateAndScanImage(context.Background(), ref, authn.Anonymous, ImageRegistryOpt{InsecureSkipTLSVerify: true})
		require.NoError(t, err)
		assert.Same(t, wantReport, got)

		require.NotNil(t, capturedCreateReq)
		assert.Equal(t, getImageManifestID(ref), capturedCreateReq.GetHashId())
		locator := capturedCreateReq.GetContainerImage()
		require.NotNil(t, locator)
		assert.True(t, locator.GetInsecureSkipTlsVerify())
		assert.Contains(t, locator.GetUrl(), ref.RepositoryStr())

		require.NotNil(t, capturedVulnReq)
		assert.Equal(t, getImageManifestID(ref), capturedVulnReq.GetHashId())
	})
}
