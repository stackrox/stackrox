package clusters

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
)

// CreateIdentity creates a new cluster identity for a service
func CreateIdentity(clusterID string, serviceType storage.ServiceType, identityStore siDataStore.DataStore) (*mtls.IssuedCert, error) {
	srvIDAllAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ServiceIdentity)))

	issuedCert, err := mtls.IssueNewCert(mtls.NewSubject(clusterID, serviceType))
	if err != nil {
		return nil, err
	}
	if err := identityStore.AddServiceIdentity(srvIDAllAccessCtx, issuedCert.ID); err != nil {
		return nil, err
	}

	return issuedCert, nil
}
