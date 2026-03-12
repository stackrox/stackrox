package tlsprofile

import (
	"context"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	tlspkg "github.com/openshift/controller-runtime-common/pkg/tls"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// apiserverClusterName is the singleton name of the APIServer cluster config resource.
const apiserverClusterName = "cluster"

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
// It returns:
//   - the TLSProfile for environment variable injection into managed workloads
//   - the raw TLSProfileSpec for configuring Go TLS directly
//
// Both return values are nil when no profile should be applied (non-OpenShift
// cluster, or TLS adherence not set to strict).
func FetchProfile(ctx context.Context, c ctrlClient.Reader, log logr.Logger) (*TLSProfile, *configv1.TLSProfileSpec) {
	apiServer := &configv1.APIServer{}
	if err := c.Get(ctx, types.NamespacedName{Name: apiserverClusterName}, apiServer); err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Error(err, "failed to read APIServer cluster config, using TLS defaults")
		}
		return nil, nil
	}

	if !shouldHonorClusterTLSProfile(apiServer.Spec.TLSAdherence) {
		return nil, nil
	}

	spec, err := tlspkg.GetTLSProfileSpec(apiServer.Spec.TLSSecurityProfile)
	if err != nil {
		log.Error(err, "failed to resolve TLS profile spec, using TLS defaults")
		return nil, nil
	}

	minVersion, err := convertMinVersion(spec.MinTLSVersion)
	if err != nil {
		log.Error(err, "unsupported TLS version in cluster profile, using TLS defaults")
		return nil, nil
	}

	return &TLSProfile{
		MinVersion:     minVersion,
		CipherSuites:   convertCiphersToIANA(spec.Ciphers),
		OpenSSLCiphers: convertCiphersToOpenSSL(spec.Ciphers),
	}, &spec
}

// shouldHonorClusterTLSProfile determines whether the cluster TLS profile
// should be enforced for a given TLSAdherencePolicy. Declared as a variable so
// callers can override it before the TLSAdherence API is available in OpenShift
// (see SetAlwaysHonorTLSProfile).
//
// TODO(ROX-32095): Replace with crypto.ShouldHonorClusterTLSProfile from
// library-go when https://github.com/openshift/library-go/pull/2114 is merged
// and available.
var shouldHonorClusterTLSProfile = func(adherence configv1.TLSAdherencePolicy) bool {
	switch adherence {
	case configv1.TLSAdherencePolicyNoOpinion,
		configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly:
		return false
	default:
		// Unknown values default to strict for forward compatibility, as
		// specified by the enhancement and to default to the more secure behavior.
		return true
	}
}

// SetAlwaysHonorTLSProfile overrides the adherence check so that the cluster
// TLS profile is always applied, regardless of the TLSAdherence field value.
//
// TODO(ROX-32095): Remove once the TLSAdherence API has landed in OpenShift.
func SetAlwaysHonorTLSProfile() {
	shouldHonorClusterTLSProfile = func(_ configv1.TLSAdherencePolicy) bool {
		return true
	}
}
