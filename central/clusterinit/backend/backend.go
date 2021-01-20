package backend

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
	"gopkg.in/yaml.v3"
)

// InitBundleWithMeta contains an init bundle alongside its meta data.
type InitBundleWithMeta struct {
	CertBundle clusters.CertBundle
	CaCert     string
	Meta       *storage.InitBundleMeta
}

func serviceTLS(cert *mtls.IssuedCert) map[string]interface{} {
	return map[string]interface{}{
		"serviceTLS": map[string]interface{}{
			"cert": string(cert.CertPEM),
			"key":  string(cert.KeyPEM),
		},
	}
}

// RenderAsYAML renders the receiver init bundle as YAML.
func (b *InitBundleWithMeta) RenderAsYAML() ([]byte, error) {
	certBundle := b.CertBundle
	sensorTLS := certBundle[storage.ServiceType_SENSOR_SERVICE]
	if sensorTLS == nil {
		return nil, errors.New("no sensor certificate in init bundle")
	}
	admissionControlTLS := certBundle[storage.ServiceType_ADMISSION_CONTROL_SERVICE]
	if admissionControlTLS == nil {
		return nil, errors.New("no admission control certificate in init bundle")
	}
	collectorTLS := certBundle[storage.ServiceType_COLLECTOR_SERVICE]
	if collectorTLS == nil {
		return nil, errors.New("no collector certificate in init bundle")
	}

	bundleMap := map[string]interface{}{
		"ca": map[string]interface{}{
			"cert": b.CaCert,
		},
		"sensor":           serviceTLS(sensorTLS),
		"collector":        serviceTLS(collectorTLS),
		"admissionControl": serviceTLS(admissionControlTLS),
	}

	bundleYaml, err := yaml.Marshal(bundleMap)
	if err != nil {
		return nil, errors.Wrap(err, "YAML marshalling of init bundle")
	}

	return bundleYaml, nil
}

var (
	// ErrInitBundleIsRevoked signals that an init bundle has been revoked.
	ErrInitBundleIsRevoked = errors.New("init bundle is revoked")
)

// Backend is the backend for the cluster-init component.
type Backend interface {
	GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error)
	Issue(ctx context.Context, name string) (*InitBundleWithMeta, error)
	Revoke(ctx context.Context, id string) error
	CheckRevoked(ctx context.Context, id string) error
}

func newBackend(store store.Store, certProvider CertificateProvider) Backend {
	return &backendImpl{
		store:        store,
		certProvider: certProvider,
	}
}

func checkAccess(ctx context.Context, access storage.Access) error {
	// we need access to the API token and service identity resources
	scopes := [][]sac.ScopeKey{
		{sac.AccessModeScopeKey(access), sac.ResourceScopeKey(resources.APIToken.GetResource())},
		{sac.AccessModeScopeKey(access), sac.ResourceScopeKey(resources.ServiceIdentity.GetResource())},
	}
	if allowed, err := sac.GlobalAccessScopeChecker(ctx).AllAllowed(ctx, scopes); err != nil {
		return errors.Wrap(err, "checking access")
	} else if !allowed {
		return errors.New("not allowed")
	}
	return nil
}
