package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/compliance/node/index"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/discovery"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/sync"
)

// mappingCachePath is where the repository-to-CPE mapping file (see
// mappingRefresher) is cached; scans always read from this file, never the
// network, so a slow or unavailable mapping endpoint never blocks or fails
// a scan directly. Hardcoded for now; make it a --mapping-cache-path flag
// later if reviewers ask for it to be configurable.
var mappingCachePath = filepath.Join(os.TempDir(), "roxagent-repo2cpe.json")

// Set via -ldflags at build time.
var agentVersion = "development" //XDef:STABLE_MAIN_VERSION

// Bounds enforced on the corresponding CLI flags in serveConfig.validate.
const (
	// minRescanInterval guards against a misconfigured, too-frequent
	// rescan cadence hammering the VM's disk; it has no effect on ACS
	// itself, only on load imposed on the scanned VM.
	minRescanInterval = 5 * time.Minute
	// maxRescanInterval caps how stale a cached report can get from a
	// misconfigured interval (e.g. years). Zero is rejected, not treated
	// as "disable periodic rescans" (can be changed in the future).
	maxRescanInterval = 7 * 24 * time.Hour

	// connDeadline bounds one connection's TLS handshake plus
	// request/response. Its range balances tolerating slow-but-legitimate
	// connections (e.g. under host resource contention) against limiting
	// how long a stalled or malicious peer can occupy the agent's single
	// in-flight-connection slot (see vsockserver.WithConnDeadline).
	minConnDeadline = 5 * time.Second
	maxConnDeadline = 5 * time.Minute
)

// ServeCmd returns the "serve" cobra subcommand for pull-mode operation.
func ServeCmd(ctx context.Context) *cobra.Command {
	var cfg serveConfig
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Scan and serve report over VSOCK (pull mode).",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe(ctx, cfg)
		},
	}
	cmd.Flags().Uint32Var(&cfg.port, "port", 818, "VSOCK port to listen on")
	cmd.Flags().StringVar(&cfg.hostPath, "host-path", "/", "Root filesystem path for indexing")
	cmd.Flags().StringVar(&cfg.repoCPEURL, "repo-cpe-url", repoToCPEMappingURL, "Repository to CPE mapping URL")
	cmd.Flags().DurationVar(&cfg.rescanInterval, "rescan-interval", 4*time.Hour,
		fmt.Sprintf("Interval between rescans (range %v-%v)", minRescanInterval, maxRescanInterval))
	cmd.Flags().DurationVar(&cfg.caFetchTimeout, "ca-fetch-timeout", 10*time.Second,
		"Timeout for each KubeVirt CA fetch attempt over VSOCK")
	cmd.Flags().DurationVar(&cfg.connDeadline, "conn-deadline", vsockserver.DefaultConnDeadline,
		fmt.Sprintf("Max time allowed for one connection's TLS handshake and request/response "+
			"(range %v-%v). Raising it tolerates slower legitimate connections (e.g. under host "+
			"resource contention) at the cost of letting a stalled or malicious peer occupy the "+
			"agent's single in-flight-connection slot for longer; lowering it does the opposite.",
			minConnDeadline, maxConnDeadline))
	return cmd
}

// serveConfig holds runServe's inputs, validated together by validate.
type serveConfig struct {
	port           uint32
	hostPath       string
	repoCPEURL     string
	rescanInterval time.Duration
	caFetchTimeout time.Duration
	connDeadline   time.Duration
}

func (c serveConfig) validate() error {
	if c.rescanInterval < minRescanInterval || c.rescanInterval > maxRescanInterval {
		return fmt.Errorf("rescan-interval must be between %v and %v (got %v)", minRescanInterval, maxRescanInterval, c.rescanInterval)
	}
	if c.caFetchTimeout <= 0 {
		return errors.New("ca-fetch-timeout must be greater than 0")
	}
	if c.connDeadline < minConnDeadline || c.connDeadline > maxConnDeadline {
		return fmt.Errorf("conn-deadline must be between %v and %v (got %v)", minConnDeadline, maxConnDeadline, c.connDeadline)
	}
	return nil
}

func runServe(ctx context.Context, cfg serveConfig) error {
	if err := cfg.validate(); err != nil {
		return err
	}

	// The mapping file must exist locally before the first scan can run:
	// scan() never fetches it itself (see mappingRefresher doc comment), so
	// this initial fetch is mandatory, not best-effort - if it fails after
	// retries, startup fails rather than running a scan against no data.
	mr := newMappingRefresher(cfg.repoCPEURL, mappingCachePath)
	if err := mr.fetchWithRetry(ctx); err != nil {
		return fmt.Errorf("initial repository-to-CPE mapping fetch: %w", err)
	}

	cache := &vsockserver.ReportCache{}
	vmRescanner := newRescanner(cache, cfg.hostPath, mappingCachePath, cfg.rescanInterval)

	report, err := scan(ctx, cfg.hostPath, mappingCachePath)
	if err != nil {
		return fmt.Errorf("initial scan: %w", err)
	}
	cache.SetReport(report, discoverFacts(cfg.hostPath))
	log.Infof("Initial scan complete, report cached. Num packages: %d", len(report.GetContents().GetPackages()))

	handler := vsockserver.NewHandler(cache, agentVersion)

	// TLS is mandatory: sensor always dials with TLS, so a plaintext agent is
	// unreachable. The KubeVirt CA (served by virt-handler on CID 2, port 1)
	// is fetched on demand, during each incoming handshake, whenever it
	// isn't already cached — see CARefresher.TLSConfig.
	// In KubeVirt's namespace-isolated VSOCK mode the CA service exists
	// only for the duration of an in-flight handshake, so it cannot be
	// warmed up independently ahead of time.
	caRefresher := vsockserver.NewCARefresher(vsockserver.WithFetchTimeout(cfg.caFetchTimeout))
	serverCert, err := selfSignedCert()
	if err != nil {
		return fmt.Errorf("generating server certificate: %w", err)
	}
	tlsCfg := caRefresher.TLSConfig(serverCert)
	log.Info("TLS enabled with KubeVirt CA (fetched on demand if not yet cached)")

	srv := vsockserver.NewServer(handler, tlsCfg, vsockserver.WithConnDeadline(cfg.connDeadline))

	ln, err := vsock.Listen(cfg.port, nil)
	if err != nil {
		return fmt.Errorf("listening on VSOCK port %d: %w", cfg.port, err)
	}
	log.Infof("Listening on VSOCK port %d (pull mode)", cfg.port)

	var wg sync.WaitGroup
	wg.Go(func() { srv.Serve(ctx, ln) })
	wg.Go(func() { vmRescanner.Run(ctx) })
	wg.Go(func() { mr.Run(ctx) })

	<-ctx.Done()
	// Wait for Serve's graceful drain (in-flight connections), the rescan
	// loop, and the mapping refresh loop to finish before returning, so the
	// process doesn't exit mid-drain, mid-scan, or mid-fetch.
	wg.Wait()
	return nil
}

// scan indexes the VM filesystem at hostPath, consulting the
// repository-to-CPE mapping data cached at mappingFilePath. It never makes
// a network call itself: mappingRefresher is solely responsible for
// keeping mappingFilePath fresh (see its doc comment for why the fetch is
// decoupled from every individual scan).
func scan(ctx context.Context, hostPath, mappingFilePath string) (*v4.IndexReport, error) {
	cfg := index.NodeIndexerConfig{
		HostPath:            hostPath,
		Client:              &http.Client{Transport: proxy.RoundTripper()},
		Repo2CPEMappingFile: mappingFilePath,
		PackageDBFilter:     "",
	}
	return index.NewNodeIndexer(cfg).IndexNode(ctx)
}

func discoverFacts(hostPath string) map[string]string {
	d := discovery.DiscoverVMData(hostPath)
	return map[string]string{
		"detected_os":         d.GetDetectedOs().String(),
		"os_version":          d.GetOsVersion(),
		"activation_status":   d.GetActivationStatus().String(),
		"dnf_metadata_status": d.GetDnfMetadataStatus().String(),
	}
}

// selfSignedCert generates a self-signed ECDSA TLS certificate.
//
// This cert exists solely to satisfy TLS protocol requirements: a server MUST
// present a certificate so the key exchange can establish an encrypted channel.
// No party in the connection path validates this cert's identity or expiry —
// virt-handler connects with InsecureSkipVerify: true and no VerifyPeerCertificate
// callback (see kubevirt/kubevirt pkg/virt-handler/rest/console.go, VSOCKHandler).
// Authentication is handled in the opposite direction: the agent verifies
// virt-handler's client cert against the KubeVirt CA via RequireAndVerifyClientCert.
//
// The cert is ephemeral: regenerated on every agent start, never persisted.
func selfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating ECDSA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating serial: %w", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("creating certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}, nil
}
