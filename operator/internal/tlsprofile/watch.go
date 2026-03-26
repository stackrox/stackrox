package tlsprofile

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	tlspkg "github.com/openshift/controller-runtime-common/pkg/tls"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupTLSProfileWatcher registers a controller that watches the
// apiserver.config.openshift.io/cluster resource. When the TLS profile or
// adherence policy changes, it cancels ctx, causing the Operator process to
// exit and be restarted by Kubernetes with the new TLS configuration applied
// to its metrics server and the updated profile propagated to managed workloads.
func SetupTLSProfileWatcher(mgr ctrl.Manager, clusterTLS ClusterTLSProfile, cancel context.CancelFunc) error {
	logger := mgr.GetLogger()
	watcher := &tlspkg.SecurityProfileWatcher{
		Client:                    mgr.GetClient(),
		InitialTLSProfileSpec:     clusterTLS.ProfileSpec,
		InitialTLSAdherencePolicy: clusterTLS.Adherence,
		OnProfileChange: func(_ context.Context, oldSpec, newSpec configv1.TLSProfileSpec) {
			logger.Info(
				"cluster TLS profile changed, restarting Operator to apply new settings",
				"oldMinTLSVersion", oldSpec.MinTLSVersion,
				"newMinTLSVersion", newSpec.MinTLSVersion,
			)
			cancel()
		},
		OnAdherencePolicyChange: func(_ context.Context, oldPolicy, newPolicy configv1.TLSAdherencePolicy) {
			logger.Info(
				"cluster TLS adherence policy changed, restarting Operator to apply new settings",
				"oldPolicy", oldPolicy,
				"newPolicy", newPolicy,
			)
			cancel()
		},
	}

	if err := watcher.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setting up TLS profile watcher: %w", err)
	}
	return nil
}
