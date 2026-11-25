package openshift

import (
	"context"

	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	authProviderID  = "b3070020-ecc3-4f34-a3f6-ad28ffc9b80f"
	permissionSetID = "b3070020-ecc3-4f34-a3f6-ad28ffc9b80e"

	// Auth provider is hidden from the login screen but visible in Access Control.
	// It trusts the Kubernetes client CA from extension-apiserver-authentication,
	// which signs Prometheus client certs. This adds the CA to userTrustRoots on the
	// public :8443 endpoint (optional client cert verification).
	authProviderName  = "OpenShift Platform Client Certificates"
	permissionSetName = "OpenShift Prometheus Metrics Reader"
)

var log = logging.LoggerForModule()

// SeedPlatformClientCertAuthProvider maintains OpenShift's platform client CA in
// the userpki provider independently of Central's serving certificate. It grants
// no roles automatically. The existing metrics permission set is retained for
// administrators and Sensor-proxied metrics.
func SeedPlatformClientCertAuthProvider(ctx context.Context, registry authproviders.Registry, roleDS roleDataStore.DataStore) {
	if !env.Openshift.BooleanSetting() {
		return
	}

	ensurePermissionSet(ctx, roleDS)

	clientset, err := newK8sClient()
	if err != nil {
		log.Warnf("Cannot create Kubernetes client for OpenShift platform certificate auth: %v", err)
		return
	}

	onCA := func(caPEM string) {
		// Auth provider uses DEFAULT origin.
		defaultCtx := declarativeconfig.WithModifyDefaultResource(ctx)
		ensureAuthProvider(defaultCtx, registry, caPEM)
	}

	if caPEM := readClientCA(ctx, clientset); caPEM != "" {
		onCA(caPEM)
	}

	watchClientCA(ctx, clientset, onCA)
}

func newK8sClient() (kubernetes.Interface, error) {
	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func readClientCA(ctx context.Context, clientset kubernetes.Interface) string {
	cm, err := clientset.CoreV1().ConfigMaps(
		env.SecureMetricsClientCANamespace.Setting(),
	).Get(ctx, env.SecureMetricsClientCAConfigMap.Setting(), metav1.GetOptions{})
	if err != nil {
		log.Warnf("Cannot read client CA ConfigMap %s/%s: %v",
			env.SecureMetricsClientCANamespace.Setting(),
			env.SecureMetricsClientCAConfigMap.Setting(), err)
		return ""
	}
	pem, ok := cm.Data[env.SecureMetricsClientCAKey.Setting()]
	if !ok || pem == "" {
		log.Warnf("Client CA key %q not found in ConfigMap %s/%s",
			env.SecureMetricsClientCAKey.Setting(),
			env.SecureMetricsClientCANamespace.Setting(),
			env.SecureMetricsClientCAConfigMap.Setting())
		return ""
	}
	return pem
}

func watchClientCA(ctx context.Context, clientset kubernetes.Interface, onCA func(string)) {
	watcher := k8scfgwatch.NewConfigMapWatcher(clientset, func(cm *v1.ConfigMap) {
		if cm == nil {
			return
		}
		pem, ok := cm.Data[env.SecureMetricsClientCAKey.Setting()]
		if !ok || pem == "" {
			return
		}
		onCA(pem)
	})
	watcher.Watch(
		ctx,
		env.SecureMetricsClientCANamespace.Setting(),
		env.SecureMetricsClientCAConfigMap.Setting(),
	)
}

func refreshCA(ctx context.Context, registry authproviders.Registry, provider authproviders.Provider, caPEM string) {
	config := provider.StorageView().GetConfig()
	if config[userpki.ConfigKeys] == caPEM {
		return
	}
	log.Info("Client CA changed, updating OpenShift platform certificate auth provider")
	config[userpki.ConfigKeys] = caPEM

	factory := provider.BackendFactory()
	if factory == nil {
		log.Warn("Auth provider has no backend factory, cannot apply the new client CA")
		return
	}
	// WithConfig on its own only rewrites the stored config. The provider keeps its
	// existing backend, which parsed the certificates once at construction time, and the
	// registry then re-registers those same stale certificates with the client CA
	// manager. Central would keep advertising the superseded CA until restart, so the
	// backend has to be rebuilt from the new config as part of the same update.
	if _, err := registry.UpdateProvider(declarativeconfig.WithModifyDefaultResource(ctx), authProviderID,
		authproviders.WithConfig(config),
		authproviders.WithBackendFromFactory(ctx, factory),
	); err != nil {
		log.Warnf("Failed to update client CA in auth provider: %v", err)
	}
}

func ensurePermissionSet(ctx context.Context, roleDS roleDataStore.DataStore) {
	if exists, err := roleDS.PermissionSetExists(ctx, permissionSetID); err != nil {
		log.Warnf("Failed to check OpenShift metrics permission set: %v", err)
		return
	} else if exists {
		return
	}

	ps := &storage.PermissionSet{
		Id:          permissionSetID,
		Name:        permissionSetName,
		Description: "For OpenShift Prometheus: use it to grant read-only access to scoped resources used by custom metrics",
		// Let's give no permissions by default to avoid unexpected sensitive data exposure.
		ResourceToAccess: permissionsUtils.FromResourcesWithAccess(),
	}
	if err := roleDS.AddPermissionSet(ctx, ps); err != nil {
		log.Warnf("Failed to create OpenShift metrics permission set: %v", err)
	}
}

func ensureAuthProvider(ctx context.Context, registry authproviders.Registry, caPEM string) {
	if existing := registry.GetProvider(authProviderID); existing != nil {
		refreshCA(ctx, registry, existing, caPEM)
		return
	}

	_, err := registry.CreateProvider(declarativeconfig.WithModifyDefaultResource(ctx),
		authproviders.WithType(userpki.TypeName),
		authproviders.WithID(authProviderID),
		authproviders.WithName(authProviderName),
		authproviders.WithEnabled(true),
		authproviders.WithActive(true),
		authproviders.WithVisibility(storage.Traits_HIDDEN),
		authproviders.WithOrigin(storage.Traits_DEFAULT),
		authproviders.WithConfig(map[string]string{
			userpki.ConfigKeys: caPEM,
		}),
	)
	if err != nil {
		log.Warnf("Failed to create OpenShift platform certificate auth provider: %v", err)
	}
}
