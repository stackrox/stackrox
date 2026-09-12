package openshift

import (
	"context"

	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	rolePkg "github.com/stackrox/rox/central/role"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/tlsconfig"
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
	roleName          = "OpenShift Prometheus Metrics Reader"
	permissionSetName = "OpenShift Prometheus Metrics Reader"
)

var log = logging.LoggerForModule()

// SeedMetricsAuthProvider ensures that the userpki auth provider, permission set, role,
// and group mapping required for OpenShift Prometheus to scrape /metrics exist.
//
// The auth provider uses the Kubernetes client CA from the
// extension-apiserver-authentication ConfigMap (same CA that signs Prometheus client
// certs). A ConfigMap watcher keeps the CA current across rotations.
//
// All seeded objects use IMPERATIVE origin. Ideally they would be DEFAULT
// (non-user-modifiable), but the auth provider datastore rejects non-IMPERATIVE
// origins through the normal creation path (see CanModifyResource).
func SeedMetricsAuthProvider(ctx context.Context, registry authproviders.Registry, roleDS roleDataStore.DataStore, groupDS groupDataStore.DataStore) {
	if !tlsconfig.OpenShiftTLSConfigured() {
		return
	}

	clientset, err := newK8sClient()
	if err != nil {
		log.Warnf("Cannot create Kubernetes client for OpenShift metrics auth: %v", err)
		return
	}

	onCA := func(caPEM string) {
		ensurePermissionSet(ctx, roleDS)
		ensureRole(ctx, roleDS)
		ensureAuthProvider(ctx, registry, caPEM)
		ensureGroup(ctx, groupDS)
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
	log.Info("Client CA changed, updating OpenShift metrics auth provider")
	config[userpki.ConfigKeys] = caPEM
	if _, err := registry.UpdateProvider(ctx, authProviderID, authproviders.WithConfig(config)); err != nil {
		log.Warnf("Failed to update client CA in auth provider: %v", err)
	}
}

func ensurePermissionSet(ctx context.Context, roleDS roleDataStore.DataStore) {
	if _, exists, err := roleDS.GetPermissionSet(ctx, permissionSetID); err != nil {
		log.Warnf("Failed to check OpenShift metrics permission set: %v", err)
		return
	} else if exists {
		return
	}

	ps := &storage.PermissionSet{
		Id:          permissionSetID,
		Name:        permissionSetName,
		Description: "For OpenShift Prometheus: read access to Administration for /metrics scraping",
		// Let's give no permissions by default to avoid unexpected sensitive data exposure.
		ResourceToAccess: permissionsUtils.FromResourcesWithAccess(),
	}
	if err := roleDS.AddPermissionSet(ctx, ps); err != nil {
		log.Warnf("Failed to create OpenShift metrics permission set: %v", err)
	}
}

func ensureRole(ctx context.Context, roleDS roleDataStore.DataStore) {
	if _, exists, err := roleDS.GetRole(ctx, roleName); err != nil {
		log.Warnf("Failed to check OpenShift metrics role: %v", err)
		return
	} else if exists {
		return
	}

	role := &storage.Role{
		Name:            roleName,
		Description:     "Maps OpenShift Prometheus service account to /metrics read access",
		PermissionSetId: permissionSetID,
		AccessScopeId:   rolePkg.AccessScopeIncludeAll.GetId(),
	}
	if err := roleDS.AddRole(ctx, role); err != nil {
		log.Warnf("Failed to create OpenShift metrics role: %v", err)
	}
}

func ensureAuthProvider(ctx context.Context, registry authproviders.Registry, caPEM string) {
	if existing := registry.GetProvider(authProviderID); existing != nil {
		refreshCA(ctx, registry, existing, caPEM)
		return
	}

	_, err := registry.CreateProvider(ctx,
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
		log.Warnf("Failed to create OpenShift metrics auth provider: %v", err)
	}
}

func ensureGroup(ctx context.Context, groupDS groupDataStore.DataStore) {
	cn := env.SecureMetricsClientCertCN.Setting()
	existing, err := groupDS.GetFiltered(ctx, func(g *storage.Group) bool {
		p := g.GetProps()
		return p.GetAuthProviderId() == authProviderID && p.GetKey() == "name" && p.GetValue() == cn
	})
	if err != nil {
		log.Warnf("Failed to check OpenShift metrics group: %v", err)
		return
	}
	if len(existing) > 0 {
		return
	}
	group := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: authProviderID,
			Key:            "name",
			Value:          cn,
			Traits: &storage.Traits{
				Origin: storage.Traits_DEFAULT,
			},
		},
		RoleName: roleName,
	}
	// Group uses DEFAULT origin so a user can't repoint or delete the mapping and thereby
	// orphan (and delete) the role out from under it.
	if err := groupDS.Add(declarativeconfig.WithModifyDefaultResource(ctx), group); err != nil {
		log.Warnf("Failed to create OpenShift metrics group: %v", err)
	}
}
