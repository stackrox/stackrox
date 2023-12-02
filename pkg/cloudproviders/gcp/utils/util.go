package utils

import (
	"context"

	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	googleStorage "cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/auth"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/handler"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/types"
	"golang.org/x/oauth2/google"
	securitycenterv1 "google.golang.org/api/securitycenter/v1"
	googleStoragev1 "google.golang.org/api/storage/v1"
)

// CreateStorageHandlerFromConfig creates a handler based on the GCS integration configuration.
func CreateStorageHandlerFromConfig(ctx context.Context, conf *storage.GCSConfig) (handler.Handler[*googleStorage.Client], error) {
	if conf.GetUseWorkloadId() {
		return createDefaultCredsHandler[*googleStorage.Client](ctx, googleStoragev1.CloudPlatformScope)
	}

	return createStaticHandler[*googleStorage.Client](ctx, []byte(conf.GetServiceAccount()),
		googleStoragev1.CloudPlatformScope)
}

// CreateStorageHandlerFromConfigWithManager creates a handler based on the GCS integration configuration.
func CreateStorageHandlerFromConfigWithManager(ctx context.Context, conf *storage.GCSConfig,
	manager auth.STSClientManager) (handler.Handler[*googleStorage.Client], error) {
	if conf.GetUseWorkloadId() {
		return manager.StorageClientHandler(), nil
	}

	return createStaticHandler[*googleStorage.Client](ctx, []byte(conf.GetServiceAccount()),
		googleStoragev1.CloudPlatformScope)
}

// CreateSecurityCenterHandlerFromConfig creates a handler based on the security center config.
func CreateSecurityCenterHandlerFromConfig(ctx context.Context, decCreds []byte,
	wifEnabled bool) (handler.Handler[*securitycenter.Client], error) {
	if wifEnabled {
		return createDefaultCredsHandler[*securitycenter.Client](ctx, securitycenterv1.CloudPlatformScope)
	}

	return createStaticHandler[*securitycenter.Client](ctx, decCreds, securitycenterv1.CloudPlatformScope)
}

// CreateSecurityCenterHandlerFromConfigWithManager creates a handler based on the security center config.
func CreateSecurityCenterHandlerFromConfigWithManager(ctx context.Context, manager auth.STSClientManager, decCreds []byte,
	wifEnabled bool) (handler.Handler[*securitycenter.Client], error) {
	if wifEnabled {
		return manager.SecurityCenterClientHandler(), nil
	}

	return createStaticHandler[*securitycenter.Client](ctx, decCreds, securitycenterv1.CloudPlatformScope)
}

func createStaticHandler[T types.GcpSDKClients](ctx context.Context, credentialBytes []byte, scope string) (handler.Handler[T], error) {
	creds, err := google.CredentialsFromJSON(ctx, credentialBytes, scope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create credentials")
	}
	return handler.NewHandler[T](ctx, creds)
}

func createDefaultCredsHandler[T types.GcpSDKClients](ctx context.Context, scope string) (handler.Handler[T], error) {
	creds, err := google.FindDefaultCredentials(ctx, scope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create credentials")
	}
	return handler.NewHandler[T](ctx, creds)
}
