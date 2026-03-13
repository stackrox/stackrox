package tlsprofile

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	tlspkg "github.com/openshift/controller-runtime-common/pkg/tls"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupTLSProfileWatcher registers a controller that watches the
// apiserver.config.openshift.io/cluster resource. When the TLS profile
// changes, it cancels ctx, causing the Operator process to exit and be
// restarted by Kubernetes with the new TLS configuration applied to its
// metrics server and the updated profile propagated to managed workloads.
//
// initialProfile is the TLS profile spec already fetched at startup via
// FetchProfile. If nil, no watcher is set up (non-OpenShift / legacy mode).
func SetupTLSProfileWatcher(mgr ctrl.Manager, initialProfile *configv1.TLSProfileSpec, cancel context.CancelFunc) error {
	if initialProfile == nil {
		return nil
	}

	watcher := &tlspkg.SecurityProfileWatcher{
		Client:                mgr.GetClient(),
		InitialTLSProfileSpec: *initialProfile,
		OnProfileChange: func(_ context.Context, oldSpec, newSpec configv1.TLSProfileSpec) {
			mgr.GetLogger().Info(
				"cluster TLS profile changed, restarting Operator to apply new settings",
				"oldMinTLSVersion", oldSpec.MinTLSVersion,
				"newMinTLSVersion", newSpec.MinTLSVersion,
			)
			cancel()
		},
	}

	if err := watcher.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setting up TLS profile watcher: %w", err)
	}
	return nil
}
