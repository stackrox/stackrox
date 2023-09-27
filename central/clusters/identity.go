package clusters

import (
	"context"

	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
)

// CertBundle contains a bundle of generated certificates for each service type
type CertBundle map[storage.ServiceType]*mtls.IssuedCert

// CreateIdentity creates a new cluster identity for a service
func CreateIdentity(clusterID string, serviceType storage.ServiceType, identityStore siDataStore.DataStore, issueOpts ...mtls.IssueCertOption) (*mtls.IssuedCert, error) {
	issuedCert, err := mtls.IssueNewCert(mtls.NewSubject(clusterID, serviceType), issueOpts...)
	if err != nil {
		return nil, err
	}
	if identityStore != nil {
		administrationAllAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Administration)))
		if err := identityStore.AddServiceIdentity(administrationAllAccessCtx, issuedCert.ID); err != nil {
			return nil, err
		}
	}

	return issuedCert, nil
}

// IssueSecuredClusterCertificates creates a bundle which contains all service certificates for a given cluster
func IssueSecuredClusterCertificates(cluster *storage.Cluster, appNamespace string, identityStore siDataStore.DataStore) (CertBundle, error) {
	certs := make(CertBundle)
	var issueOpts []mtls.IssueCertOption
	if appNamespace != "" && appNamespace != namespaces.StackRox {
		issueOpts = append(issueOpts, mtls.WithNamespace(appNamespace))
	}
	for _, serviceType := range centralsensor.AllSecuredClusterServices {
		issuedCert, err := CreateIdentity(cluster.GetId(), serviceType, identityStore, issueOpts...)
		if err != nil {
			return certs, err
		}
		certs[serviceType] = issuedCert
	}
	return certs, nil
}

// IssueSecuredClusterInitCertificates creates a cert bundle which holds init certificates which have a UUID nil cluster id
// These certificates are used to register new clusters at central.
// All certificates share the same init bundle UUID written in the OU subject field.
func IssueSecuredClusterInitCertificates(tenantID string) (CertBundle, uuid.UUID, error) {
	initID := centralsensor.RegisteredInitCertClusterID
	certs := make(CertBundle)
	bundleID := uuid.NewV4()
	for _, serviceType := range centralsensor.AllSecuredClusterServices {
		issuedCert, err := mtls.IssueNewCert(mtls.NewInitSubject(initID, serviceType, bundleID, tenantID))

		if err != nil {
			return certs, bundleID, err
		}
		certs[serviceType] = issuedCert
	}
	return certs, bundleID, nil
}
