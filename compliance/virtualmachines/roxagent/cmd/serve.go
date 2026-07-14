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
	"time"

	"github.com/mdlayher/vsock"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/compliance/node/index"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/discovery"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

// Set via -ldflags at build time.
var agentVersion = "development" //XDef:STABLE_MAIN_VERSION

const mappingClientTimeout = 30 * time.Second

// minRescanInterval guards against a misconfigured, too-frequent rescan
// cadence hammering the VM's disk; it has no effect on ACS itself, only on
// load imposed on the scanned VM.
const minRescanInterval = 5 * time.Minute

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
		fmt.Sprintf("Interval between rescans (minimum %v)", minRescanInterval))
	cmd.Flags().DurationVar(&cfg.caFetchTimeout, "ca-fetch-timeout", 10*time.Second,
		"Timeout for each KubeVirt CA fetch attempt over VSOCK")
	return cmd
}

// serveConfig holds runServe's inputs, validated together by validate.
type serveConfig struct {
	port           uint32
	hostPath       string
	repoCPEURL     string
	rescanInterval time.Duration
	caFetchTimeout time.Duration
}

func (c serveConfig) validate() error {
	if c.rescanInterval < minRescanInterval {
		return fmt.Errorf("rescan-interval must be at least %v (got %v)", minRescanInterval, c.rescanInterval)
	}
	if c.caFetchTimeout <= 0 {
		return errors.New("ca-fetch-timeout must be greater than 0")
	}
	return nil
}

func runServe(ctx context.Context, cfg serveConfig) error {
	if err := cfg.validate(); err != nil {
		return err
	}

	cache := &vsockserver.ReportCache{}

	report, err := scan(ctx, cfg.hostPath, cfg.repoCPEURL)
	if err != nil {
		return fmt.Errorf("initial scan: %w", err)
	}
	cache.SetReport(report, discoverFacts(cfg.hostPath))
	log.Infof("Initial scan complete, report cached. Num packages: %d", len(report.GetContents().GetPackages()))

	handler := vsockserver.NewHandler(cache, agentVersion)

	// TLS is mandatory: sensor always dials with TLS, so a plaintext agent is
	// unreachable. The KubeVirt CA (served by virt-handler on CID 2, port 1)
	// is fetched on demand, during each incoming handshake, whenever it
	// isn't already cached — see CARefresher.TLSConfig. This is required,
	// not just an optimization: in KubeVirt's namespace-isolated VSOCK mode
	// the CA service exists only for the duration of an in-flight handshake,
	// so it cannot be warmed up independently ahead of time. Start's
	// best-effort warm-up below therefore has no error return and its
	// failure must never block startup: it's purely a latency optimization
	// for KubeVirt's "global" VSOCK mode, where the service is permanently
	// available.
	refresher := vsockserver.NewCARefresher(vsockserver.WithFetchTimeout(cfg.caFetchTimeout))
	go refresher.Start(ctx)
	serverCert, err := selfSignedCert()
	if err != nil {
		return fmt.Errorf("generating server certificate: %w", err)
	}
	tlsCfg := refresher.TLSConfig()
	tlsCfg.Certificates = []tls.Certificate{serverCert}
	log.Info("TLS enabled with KubeVirt CA (fetched on demand if not yet cached)")

	srv := vsockserver.NewServer(handler, tlsCfg)

	ln, err := vsock.Listen(cfg.port, nil)
	if err != nil {
		return fmt.Errorf("listening on VSOCK port %d: %w", cfg.port, err)
	}
	log.Infof("Listening on VSOCK port %d (pull mode)", cfg.port)

	serveDone := make(chan struct{})
	go func() {
		defer close(serveDone)
		srv.Serve(ctx, ln)
	}()

	ticker := time.NewTicker(cfg.rescanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Wait for Serve's graceful drain (in-flight connections) to finish
			// before returning, so the process doesn't exit mid-drain.
			<-serveDone
			return nil
		case <-ticker.C:
			log.Info("Starting periodic rescan")
			r, err := scan(ctx, cfg.hostPath, cfg.repoCPEURL)
			if err != nil {
				log.Errorf("Rescan failed: %v", err)
				continue
			}
			cache.SetReport(r, discoverFacts(cfg.hostPath))
			log.Infof("Rescan complete, report updated. Num packages: %d", len(r.GetContents().GetPackages()))
		}
	}
}

func scan(ctx context.Context, hostPath, repoCPEURL string) (*v4.IndexReport, error) {
	cfg := index.NodeIndexerConfig{
		HostPath:           hostPath,
		Client:             &http.Client{Transport: proxy.RoundTripper()},
		Repo2CPEMappingURL: repoCPEURL,
		Timeout:            mappingClientTimeout,
		PackageDBFilter:    "",
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
