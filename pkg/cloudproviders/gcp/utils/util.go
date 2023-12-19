package utils

import (
	"context"

	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	googleStorage "cloud.google.com/go/storage"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/auth"
	"google.golang.org/api/option"
)

// CreateStorageClientFromConfig creates a client based on the GCS integration configuration.
func CreateStorageClientFromConfig(ctx context.Context,
	conf *storage.GCSConfig,
) (*googleStorage.Client, error) {
	if conf.GetUseWorkloadId() {
		return googleStorage.NewClient(ctx)
	}
	return googleStorage.NewClient(ctx, option.WithCredentialsJSON([]byte(conf.GetServiceAccount())))
}

// CreateStorageClientFromConfigWithManager creates a client based on the GCS integration configuration.
func CreateStorageClientFromConfigWithManager(ctx context.Context,
	conf *storage.GCSConfig, manager auth.STSTokenManager,
) (*googleStorage.Client, error) {
	if conf.GetUseWorkloadId() {
		return googleStorage.NewClient(ctx, option.WithTokenSource(manager.TokenSource()))
	}
	return googleStorage.NewClient(ctx, option.WithCredentialsJSON([]byte(conf.GetServiceAccount())))
}

// CreateSecurityCenterClientFromConfig creates a client based on the security center config.
func CreateSecurityCenterClientFromConfig(ctx context.Context,
	decCreds []byte, wifEnabled bool,
) (*securitycenter.Client, error) {
	if wifEnabled {
		return securitycenter.NewClient(ctx)
	}
	return securitycenter.NewClient(ctx, option.WithCredentialsJSON([]byte(decCreds)))
}

// CreateSecurityCenterClientFromConfigWithManager creates a client based on the security center config.
func CreateSecurityCenterClientFromConfigWithManager(ctx context.Context,
	manager auth.STSTokenManager, decCreds []byte, wifEnabled bool,
) (*securitycenter.Client, error) {
	if wifEnabled {
		return securitycenter.NewClient(ctx, option.WithTokenSource(manager.TokenSource()))
	}
	return securitycenter.NewClient(ctx, option.WithCredentialsJSON([]byte(decCreds)))
}
