package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scannerv4"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	errIndexerNotConfigured = errors.New("indexer not configured")
	errMatcherNotConfigured = errors.New("matcher not configured")
)

// callOptions contains optional data and gRPC parameters for the underlying
// Scanner calls.
type callOptions struct {
	version                     *scannerv4.Version
	includeExternalIndexReports bool
}

// CallOption configures call-specific options for scanner methods.
type CallOption func(*callOptions)

// makeCallOptions processes all passed CallOptions to callOptions.
func makeCallOptions(callOpts ...CallOption) callOptions {
	var options callOptions
	for _, callOpt := range callOpts {
		callOpt(&options)
	}
	return options
}

// Version returns a CallOption that captures service version metadata.
func Version(v *scannerv4.Version) CallOption {
	return func(o *callOptions) {
		o.version = v
	}
}

// IncludeExternalIndexReports returns a CallOption that will inform library
// calls to include external index reports when retrieving index reports from
// Scanner V4's Indexer.
func IncludeExternalIndexReports() CallOption {
	return func(o *callOptions) {
		o.includeExternalIndexReports = true
	}
}

// Scanner is the interface that contains the StackRox Scanner
// application-oriented methods. It's offered to simplify application code to
// call StackRox Scanner.
//
//go:generate mockgen-wrapper
type Scanner interface {
	// GetImageIndex fetches an existing index report for the given ID.
	GetImageIndex(ctx context.Context, hashID string, callOpts ...CallOption) (*v4.IndexReport, bool, error)

	// GetOrCreateImageIndex first attempts to get an existing index report for the
	// image reference, and if not found or invalid, it then attempts to index the
	// image and return the generated index report if successful, or error.
	GetOrCreateImageIndex(ctx context.Context, ref name.Digest, auth authn.Authenticator, opt ImageRegistryOpt, callOpts ...CallOption) (*v4.IndexReport, error)

	// IndexAndScanImage scans an image for vulnerabilities. If the index report
	// for that image does not exist, it is created. It returns the vulnerability
	// report.
	IndexAndScanImage(context.Context, name.Digest, authn.Authenticator, ImageRegistryOpt, ...CallOption) (*v4.VulnerabilityReport, error)

	// GetVulnerabilities will match vulnerabilities to the contents provided.
	GetVulnerabilities(ctx context.Context, ref name.Digest, contents *v4.Contents, callOpts ...CallOption) (*v4.VulnerabilityReport, error)

	// GetMatcherMetadata returns metadata from the matcher.
	GetMatcherMetadata(context.Context, ...CallOption) (*v4.Metadata, error)

	// GetSBOM to get sbom for an image
	GetSBOM(ctx context.Context, name string, ref name.Digest, uri string, callOpts ...CallOption) ([]byte, bool, error)

	// StoreImageIndex stores the contents provided. Particularly useful for
	// storing contents from delegated Scanners. indexerVersion is used to
	// hint to the Scanner whether it should overwrite the contents of ref
	// if ref already exists in its datastore.
	StoreImageIndex(ctx context.Context, ref name.Digest, indexerVersion string, contents *v4.Contents, callOpts ...CallOption) error

	// Close cleans up any resources used by the implementation.
	Close() error
}

// gRPCScanner A scanner client implementation based on gRPC endpoints.
type gRPCScanner struct {
	indexer         v4.IndexerClient
	matcher         v4.MatcherClient
	gRPCConnections []*grpc.ClientConn
}

// NewGRPCScanner creates a new gRPC scanner client.
func NewGRPCScanner(ctx context.Context, opts ...Option) (Scanner, error) {
	o, err := makeOptions(opts...)
	if err != nil {
		return nil, err
	}

	if o.comboMode {
		// Both o.indexerOpts and o.matcherOpts are the same, so just choose one.
		conn, err := createGRPCConn(ctx, o.indexerOpts)
		if err != nil {
			return nil, err
		}
		return &gRPCScanner{
			gRPCConnections: []*grpc.ClientConn{conn},
			indexer:         v4.NewIndexerClient(conn),
			matcher:         v4.NewMatcherClient(conn),
		}, nil
	}

	var success bool
	conns := make([]*grpc.ClientConn, 0, 2)
	defer func() {
		if !success {
			for _, conn := range conns {
				utils.IgnoreError(conn.Close)
			}
		}
	}()

	var indexerClient v4.IndexerClient
	if o.indexerOpts.address != "" {
		conn, err := createGRPCConn(ctx, o.indexerOpts)
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
		indexerClient = v4.NewIndexerClient(conn)
	}

	var matcherClient v4.MatcherClient
	if o.matcherOpts.address != "" {
		conn, err := createGRPCConn(ctx, o.matcherOpts)
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
		matcherClient = v4.NewMatcherClient(conn)
	}

	success = true
	return &gRPCScanner{
		gRPCConnections: conns,
		indexer:         indexerClient,
		matcher:         matcherClient,
	}, nil
}

// Close closes the gRPC connection.
func (c *gRPCScanner) Close() error {
	errList := errorhelpers.NewErrorList("closing connections")
	for _, conn := range c.gRPCConnections {
		errList.AddError(conn.Close())
	}
	return errList.ToError()
}

func createGRPCConn(ctx context.Context, o connOptions) (*grpc.ClientConn, error) {
	// Prefix address with dns:/// to use the DNS name resolver.
	address := "dns:///" + o.address

	dialOpts := []grpc.DialOption{
		// Scanner v4 Indexer and Matcher pods are accessed via gRPC, which Kubernetes Services
		// cannot properly load balance. Kubernetes Service load balancer is connection-based
		// (best for HTTP/1.x), while gRPC is built on top of HTTP/2 which tends to just use a single connection
		// for all requests.
		//
		// We opt to do client-side load balancing, instead,
		// via DNS name resolution, which is possible because Scanner v4 services are "headless"
		// (clusterIP: None).
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin": {}}]}`),
		// The gRPC library does not respect NO_PROXY settings when using the DNS name resolver.
		// Outside from testing, the only users of this client library are Central and Sensor,
		// which only communicate with namespace-local Scanner v4 services.
		// Because of this, we just disable proxy settings when talking to Scanner v4 services.
		//
		// See https://github.com/grpc/grpc-go/issues/3401 for more information about using
		// proxy settings with DNS name resolution.
		grpc.WithNoProxy(),
	}

	maxRespMsgSize := env.ScannerV4MaxRespMsgSize.IntegerSetting()

	if o.skipTLSVerify {
		connOpts := clientconn.Options{
			TLS: clientconn.TLSConfigOptions{
				GRPCOnly:           true,
				InsecureSkipVerify: true,
			},
			DialOptions: dialOpts,
		}
		callOpts := []grpc.CallOption{grpc.MaxCallRecvMsgSize(maxRespMsgSize)}
		return clientconn.GRPCConnection(ctx, o.mTLSSubject, address, connOpts, grpc.WithDefaultCallOptions(callOpts...))
	}

	connOpts := []clientconn.ConnectionOption{
		clientconn.ServerName(o.serverName),
		clientconn.MaxMsgReceiveSize(maxRespMsgSize),
		clientconn.WithDialOptions(dialOpts...),
	}
	return clientconn.AuthenticatedGRPCConnection(ctx, address, o.mTLSSubject, connOpts...)
}

// GetSBOM verifies that index report exists and calls matcher to return sbom for an image
func (c *gRPCScanner) GetSBOM(ctx context.Context, imageFullName string, ref name.Digest, uri string, callOpts ...CallOption) ([]byte, bool, error) {
	options := makeCallOptions(callOpts...)

	// verify index report exists for the image
	hashID := getImageManifestID(ref)
	ir, found, err := c.getImageIndex(ctx, hashID, options)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}

	gsbomr := &v4.GetSBOMRequest{}
	gsbomr.SetId(ref.DigestStr())
	gsbomr.SetName(imageFullName)
	gsbomr.SetUri(uri)
	gsbomr.SetContents(ir.GetContents())
	resp, err := c.matcher.GetSBOM(ctx, gsbomr)
	return resp.GetSbom(), true, err
}

// GetImageIndex calls the Indexer's gRPC endpoint GetIndexReport.
func (c *gRPCScanner) GetImageIndex(ctx context.Context, hashID string, callOpts ...CallOption) (*v4.IndexReport, bool, error) {
	if c.indexer == nil {
		return nil, false, errIndexerNotConfigured
	}

	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/client",
		"method", "GetImageIndex",
		"hash_id", hashID,
	)

	options := makeCallOptions(callOpts...)

	return c.getImageIndex(ctx, hashID, options)
}

// GetOrCreateImageIndex calls the Indexer's gRPC endpoint GetOrCreateIndexReport.
func (c *gRPCScanner) GetOrCreateImageIndex(ctx context.Context, ref name.Digest, auth authn.Authenticator, opt ImageRegistryOpt, callOpts ...CallOption) (*v4.IndexReport, error) {
	if c.indexer == nil {
		return nil, errIndexerNotConfigured
	}

	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/client",
		"method", "GetOrCreateImageIndex",
		"image", ref.String(),
	)

	options := makeCallOptions(callOpts...)

	return c.getOrCreateImageIndex(ctx, ref, auth, opt, options)
}

// IndexAndScanImage gets or creates an index report for the image, then call the
// matcher to return a vulnerability report.
func (c *gRPCScanner) IndexAndScanImage(ctx context.Context, ref name.Digest, auth authn.Authenticator, opt ImageRegistryOpt, callOpts ...CallOption) (*v4.VulnerabilityReport, error) {
	if c.indexer == nil {
		return nil, errIndexerNotConfigured
	}
	if c.matcher == nil {
		return nil, errMatcherNotConfigured
	}

	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/client",
		"method", "IndexAndScanImage",
		"image", ref.String(),
	)

	options := makeCallOptions(callOpts...)

	ir, err := c.getOrCreateImageIndex(ctx, ref, auth, opt, options)
	if err != nil {
		return nil, fmt.Errorf("get or create index: %w", err)
	}

	return c.getVulnerabilities(ctx, ir.GetHashId(), nil, options)
}

func (c *gRPCScanner) getImageIndex(ctx context.Context, hashID string, options callOptions) (*v4.IndexReport, bool, error) {
	req := &v4.GetIndexReportRequest{}
	req.SetHashId(hashID)
	req.SetIncludeExternal(options.includeExternalIndexReports)
	var ir *v4.IndexReport
	var responseMetadata metadata.MD
	// Get the IndexReport, if it exists.
	err := retryWithBackoff(ctx, defaultBackoff(), "indexer.GetIndexReport", func() (err error) {
		ir, err = c.indexer.GetIndexReport(ctx, req, grpc.Header(&responseMetadata))
		if e, ok := status.FromError(err); ok && e.Code() == codes.NotFound {
			return nil
		}
		return err
	})
	if err != nil {
		return nil, false, fmt.Errorf("get index: %w", err)
	}
	// Return not found if report doesn't exist or is unsuccessful.
	if !ir.GetSuccess() {
		return nil, false, nil
	}

	setIndexerVersion(options, responseMetadata)

	return ir, true, nil
}

func (c *gRPCScanner) getOrCreateImageIndex(ctx context.Context, ref name.Digest, auth authn.Authenticator, opt ImageRegistryOpt, options callOptions) (*v4.IndexReport, error) {
	id := getImageManifestID(ref)
	imgURL := &url.URL{
		Scheme: ref.Context().Scheme(),
		Host:   ref.RegistryStr(),
		Path:   fmt.Sprintf("%s@%s", ref.RepositoryStr(), ref.DigestStr()),
	}
	authCfg, err := auth.Authorization()
	if err != nil {
		return nil, fmt.Errorf("get auth: %w", err)
	}
	cil := &v4.ContainerImageLocator{}
	cil.SetUrl(imgURL.String())
	cil.SetUsername(authCfg.Username)
	cil.SetPassword(authCfg.Password)
	cil.SetInsecureSkipTlsVerify(opt.InsecureSkipTLSVerify)
	req := &v4.GetOrCreateIndexReportRequest{}
	req.SetHashId(id)
	req.SetContainerImage(proto.ValueOrDefault(cil))
	var ir *v4.IndexReport
	var responseMetadata metadata.MD
	err = retryWithBackoff(ctx, defaultBackoff(), "indexer.GetOrCreateIndexReport", func() (err error) {
		ir, err = c.indexer.GetOrCreateIndexReport(ctx, &req, grpc.Header(&responseMetadata))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}

	setIndexerVersion(options, responseMetadata)

	return ir, nil
}

func (c *gRPCScanner) GetVulnerabilities(ctx context.Context, ref name.Digest, contents *v4.Contents, callOpts ...CallOption) (*v4.VulnerabilityReport, error) {
	if c.matcher == nil {
		return nil, errMatcherNotConfigured
	}

	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/client",
		"method", "GetVulnerabilities",
		"image", ref.String(),
	)

	options := makeCallOptions(callOpts...)

	return c.getVulnerabilities(ctx, getImageManifestID(ref), contents, options)
}

func (c *gRPCScanner) getVulnerabilities(ctx context.Context, hashID string, contents *v4.Contents, options callOptions) (*v4.VulnerabilityReport, error) {
	req := &v4.GetVulnerabilitiesRequest{}
	req.SetHashId(hashID)
	req.SetContents(contents)
	var vr *v4.VulnerabilityReport
	var responseMetadata metadata.MD
	err := retryWithBackoff(ctx, defaultBackoff(), "matcher.GetVulnerabilities", func() (err error) {
		vr, err = c.matcher.GetVulnerabilities(ctx, req, grpc.Header(&responseMetadata))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get vulns: %w", err)
	}

	setMatcherVersion(options, responseMetadata)

	return vr, nil
}

func (c *gRPCScanner) GetMatcherMetadata(ctx context.Context, callOpts ...CallOption) (*v4.Metadata, error) {
	if c.matcher == nil {
		return nil, errMatcherNotConfigured
	}

	ctx = zlog.ContextWithValues(ctx, "component", "scanner/client", "method", "GetMatcherMetadata")

	options := makeCallOptions(callOpts...)

	var m *v4.Metadata
	var responseMetadata metadata.MD
	err := retryWithBackoff(ctx, defaultBackoff(), "matcher.GetMetadata", func() error {
		var err error
		m, err = c.matcher.GetMetadata(ctx, protocompat.ProtoEmpty(), grpc.Header(&responseMetadata))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	setMatcherVersion(options, responseMetadata)

	return m, nil
}

// StoreImageIndex calls the Indexer's gRPC endpoint StoreIndexReport.
// The ...CallOption is included for consistency but isn't currently used.
func (c *gRPCScanner) StoreImageIndex(ctx context.Context, ref name.Digest, indexerVersion string, contents *v4.Contents, _ ...CallOption) error {
	if c.indexer == nil {
		return errIndexerNotConfigured
	}

	ctx = zlog.ContextWithValues(ctx, "component", "scanner/client", "method", "StoreImageIndex")

	req := &v4.StoreIndexReportRequest{}
	req.SetHashId(getImageManifestID(ref))
	req.SetIndexerVersion(indexerVersion)
	req.SetContents(contents)
	var r *v4.StoreIndexReportResponse
	err := retryWithBackoff(ctx, defaultBackoff(), "indexer.StoreImageIndex", func() error {
		var err error
		r, err = c.indexer.StoreIndexReport(ctx, req)
		return err
	})
	if err != nil {
		return fmt.Errorf("storing external index report: %w", err)
	}
	zlog.Debug(ctx).Err(err).Str("status", r.GetStatus()).Msg("received response from StoreIndexReport")

	return nil
}

func getImageManifestID(ref name.Digest) string {
	return fmt.Sprintf("/v4/containerimage/%s", ref.DigestStr())
}

// retryWithBackoff is a utility function to wrap backoff.Retry to handle common
// retryable gRPC codes.
func retryWithBackoff(ctx context.Context, b backoff.BackOff, rpc string, op backoff.Operation) error {
	ctx = zlog.ContextWithValues(ctx, "rpc", rpc)
	f := func() error {
		err := op()
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.OK:
				return nil
			case codes.Aborted, codes.Unavailable, codes.Internal:
			default:
				return backoff.Permanent(err)
			}
		}
		return err
	}
	return backoff.RetryNotify(f, backoff.WithContext(b, ctx), func(err error, duration time.Duration) {
		zlog.Debug(ctx).Err(err).Dur("duration", duration).Msg("retrying gRPC call")
	})
}

func defaultBackoff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = time.Second
	b.RandomizationFactor = 0.25
	b.MaxInterval = time.Second * 5
	b.Multiplier = 2
	b.MaxElapsedTime = time.Second * 10
	return b
}

// setIndexerVersion extracts the indexer version from the gRPC response
// metadata response. Overwrites the stored version in options if called
// more than once.
func setIndexerVersion(options callOptions, responseMetadata metadata.MD) {
	if options.version == nil {
		return
	}

	options.version.Indexer = scannerv4.DefaultVersion
	if versions := responseMetadata.Get(scannerv4.ServiceVersionHeader); len(versions) > 0 {
		options.version.Indexer = versions[0]
	}
}

// setMatcherVersion extracts the matcher version from the gRPC response
// metadata response. Overwrites the stored version in options if called
// more than once.
func setMatcherVersion(options callOptions, responseMetadata metadata.MD) {
	if options.version == nil {
		return
	}

	options.version.Matcher = scannerv4.DefaultVersion
	if versions := responseMetadata.Get(scannerv4.ServiceVersionHeader); len(versions) > 0 {
		options.version.Matcher = versions[0]
	}
}
