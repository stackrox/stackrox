package tlsprofile

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	tlspkg "github.com/openshift/controller-runtime-common/pkg/tls"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("tlsprofile")

// apiserverClusterName is the singleton name of the APIServer cluster config resource.
const apiserverClusterName = "cluster"

// ClusterTLSProfile holds the TLS profile settings read from the
// apiserver.config.openshift.io/cluster resource. A nil *ClusterTLSProfile
// means the Operator is not running on OpenShift.
type ClusterTLSProfile struct {
	// ProfileSpec is the concrete TLS profile spec (ciphers + min version).
	ProfileSpec configv1.TLSProfileSpec
	// Adherence is the current TLS adherence policy from the APIServer resource.
	Adherence configv1.TLSAdherencePolicy
}

// TLSProfile holds the parsed TLS profile settings in all formats needed
// by the various ACS components.
type TLSProfile struct {
	// MinVersion is the minimum TLS version in OpenSSL format (e.g. "TLSv1.2"),
	// which is what ROX_TLS_MIN_VERSION and PostgreSQL ssl_min_protocol_version expect.
	MinVersion string
	// CipherSuites is a comma-separated list of IANA cipher suite names
	// for ROX_TLS_CIPHER_SUITES.
	CipherSuites string
	// OpenSSLCiphers is a colon-separated OpenSSL cipher string
	// for ROX_OPENSSL_TLS_CIPHER_SUITES.
	OpenSSLCiphers string
}

// FetchProfile reads the cluster TLS profile from the APIServer resource.
//
// Returns nil on non-OpenShift clusters (APIServer resource absent or
// config.openshift.io API group not available).
// On OpenShift, ProfileSpec and Adherence are populated so the watcher can
// detect changes even when enforcement is not active yet.
func FetchProfile(ctx context.Context, c ctrlClient.Reader) (*ClusterTLSProfile, error) {
	apiServer := &configv1.APIServer{}
	if err := c.Get(ctx, types.NamespacedName{Name: apiserverClusterName}, apiServer); err != nil {
		if k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) || discovery.IsGroupDiscoveryFailedError(err) {
			log.Info("APIServer cluster config not available, using TLS defaults")
			return nil, nil
		}
		return nil, fmt.Errorf("reading APIServer cluster config: %w", err)
	}

	adherence := apiServer.Spec.TLSAdherence
	spec, err := tlspkg.GetTLSProfileSpec(apiServer.Spec.TLSSecurityProfile)
	if err != nil {
		return nil, fmt.Errorf("getting TLS profile spec: %w", err)
	}

	return &ClusterTLSProfile{ProfileSpec: spec, Adherence: adherence}, nil
}

// ConvertProfile converts a ClusterTLSProfile to a TLSProfile for environment
// variable injection into managed workloads.
//
// Returns nil when:
//   - clusterTLS is nil (not on OpenShift)
//   - forceProfile is false and the adherence policy does not require honoring
//     the cluster TLS profile
func ConvertProfile(clusterTLS *ClusterTLSProfile, forceProfile bool) *TLSProfile {
	if clusterTLS == nil {
		return nil
	}

	honorProfile := forceProfile || libgocrypto.ShouldHonorClusterTLSProfile(clusterTLS.Adherence)
	if !honorProfile {
		return nil
	}

	minVersion, known := convertMinVersion(clusterTLS.ProfileSpec.MinTLSVersion)
	if !known {
		log.Info("Unsupported TLS version in cluster profile, clamping to highest known version.",
			"requestedVersion", clusterTLS.ProfileSpec.MinTLSVersion,
			"clampedVersion", minVersion)
	}

	return &TLSProfile{
		MinVersion:     minVersion,
		CipherSuites:   convertCiphersToIANA(clusterTLS.ProfileSpec.Ciphers),
		OpenSSLCiphers: convertCiphersToOpenSSL(clusterTLS.ProfileSpec.Ciphers),
	}
}
