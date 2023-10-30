package client

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Scanner is the interface that contains the StackRox Scanner
// application-oriented methods. It's offered to simplify application code to
// call StackRox Scanner.
type Scanner interface {
	// GetOrCreateImageIndex first attempts to get an existing index report for the
	// image reference, and if not found or invalid, it then attempts to index the
	// image and return the generated index report if successful, or error.
	GetOrCreateImageIndex(ctx context.Context, ref name.Digest, auth authn.Authenticator) (*v4.IndexReport, error)

	// IndexAndScanImage scans an image for vulnerabilities. If the index report
	// for that image does not exist, it is created. It returns the vulnerability
	// report.
	IndexAndScanImage(context.Context, name.Digest, authn.Authenticator) (*v4.VulnerabilityReport, error)

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
	o := makeOptions(opts...)
	var connList []*grpc.ClientConn
	conn, err := createGRPCConn(ctx, o.indexerOpts)
	if err != nil {
		return nil, err
	}
	iConn, mConn := conn, conn
	connList = append(connList, conn)
	if !o.comboMode {
		mConn, err = createGRPCConn(ctx, o.matcherOpts)
		if err != nil {
			utils.IgnoreError(conn.Close)
			return nil, err
		}
		connList = append(connList, mConn)
	}
	indexerClient := v4.NewIndexerClient(mConn)
	matcherClient := v4.NewMatcherClient(iConn)
	return &gRPCScanner{
		gRPCConnections: connList,
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
	connOpt := clientconn.Options{
		TLS: clientconn.TLSConfigOptions{
			GRPCOnly:           true,
			InsecureSkipVerify: true,
		},
	}
	if o.skipTLS {
		ca, err := mtls.LoadDefaultCA()
		if err != nil {
			return nil, fmt.Errorf("creating CA: %w", err)
		}
		connOpt.TLS.InsecureSkipVerify = false
		connOpt.TLS.RootCAs = ca.CertPool()
		connOpt.TLS.UseClientCert = clientconn.MustUseClientCert
		connOpt.TLS.ServerName = o.serverName
	}
	return clientconn.GRPCConnection(ctx, o.mTLSSubject, o.address, connOpt)
}

// GetOrCreateImageIndex calls the Indexer's gRPC endpoint to first
// GetIndexReport, then if not found or if the report is not successful, then
// call CreateIndexReport.
func (c *gRPCScanner) GetOrCreateImageIndex(ctx context.Context, ref name.Digest, auth authn.Authenticator) (*v4.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/client", "method", "GetOrCreateImageIndex")
	id := getImageManifestID(ref)
	var ir *v4.IndexReport
	// Get the IndexReport if it exists.
	err := retryWithBackoff(ctx, defaultBackoff(), "indexer.GetIndexReport", func() (err error) {
		ir, err = c.indexer.GetIndexReport(ctx, &v4.GetIndexReportRequest{HashId: id})
		if e, ok := status.FromError(err); ok {
			if e.Code() == codes.NotFound {
				return nil
			}
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get index: %w", err)
	}
	// Returns if found and report is successful.
	if ir != nil && ir.GetSuccess() {
		return ir, nil
	}
	// Otherwise (re-)index the image.
	imgURL := &url.URL{
		Scheme: ref.Context().Scheme(),
		Host:   ref.RegistryStr(),
		Path:   fmt.Sprintf("%s@%s", ref.RepositoryStr(), ref.DigestStr()),
	}
	authCfg, err := auth.Authorization()
	if err != nil {
		return nil, fmt.Errorf("get auth: %w", err)
	}
	req := v4.CreateIndexReportRequest{
		HashId: id,
		ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{
			ContainerImage: &v4.ContainerImageLocator{
				Url:      imgURL.String(),
				Username: authCfg.Username,
				Password: authCfg.Password,
			},
		},
	}
	err = retryWithBackoff(ctx, defaultBackoff(), "indexer.CreateIndexReport", func() (err error) {
		ir, err = c.indexer.CreateIndexReport(ctx, &req)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}
	return ir, nil
}

// IndexAndScanImage get or create an index report for the image, then call the
// matcher to return a vulnerability report.
func (c *gRPCScanner) IndexAndScanImage(ctx context.Context, ref name.Digest, auth authn.Authenticator) (*v4.VulnerabilityReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/client", "method", "IndexAndScanImage")
	ir, err := c.GetOrCreateImageIndex(ctx, ref, auth)
	if err != nil {
		return nil, fmt.Errorf("get or create index: %w", err)
	}
	req := &v4.GetVulnerabilitiesRequest{HashId: ir.GetHashId()}
	var vr *v4.VulnerabilityReport
	err = retryWithBackoff(ctx, defaultBackoff(), "matcher.GetVulnerabilities", func() (err error) {
		vr, err = c.matcher.GetVulnerabilities(ctx, req)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get vulns: %w", err)
	}
	return vr, nil
}

func getImageManifestID(ref name.Digest) string {
	return fmt.Sprintf("/v4/containerimage/%s", ref.DigestStr())
}

// retryWithBackoff is a utility function to wrap backoff.Retry to handle common
// retryable gRPC codes.
func retryWithBackoff(ctx context.Context, b backoff.BackOff, rpc string, op backoff.Operation) error {
	zlog.ContextWithValues(ctx, "rpc", rpc)
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
