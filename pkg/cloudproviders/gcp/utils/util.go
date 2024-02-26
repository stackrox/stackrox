package utils

import (
	"context"
	"time"

	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	googleStorage "cloud.google.com/go/storage"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/auth"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

const earlyExpiry = 5 * time.Minute

// CreateStorageClientFromConfig creates a client based on the GCS integration configuration.
//
// We do not use proxy.RoundTripper() here because because overwriting the GCP http client
// with a custom transport causes high latency by the google SDK.
func CreateStorageClientFromConfig(ctx context.Context,
	conf *storage.GCSConfig,
) (*googleStorage.Client, error) {
	if conf.GetUseWorkloadId() {
		return googleStorage.NewClient(ctx)
	}
	return googleStorage.NewClient(ctx, option.WithCredentialsJSON([]byte(conf.GetServiceAccount())))
}

// CreateStorageClientFromConfigWithManager creates a client based on the GCS integration configuration.
//
// We do not use proxy.RoundTripper() here because because overwriting the GCP http client
// with a custom transport causes high latency by the google SDK.
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
		return securitycenter.NewClient(ctx, option.WithGRPCDialOption(grpc.WithContextDialer(proxy.AwareDialContext)))
	}
	return securitycenter.NewClient(ctx,
		option.WithCredentialsJSON([]byte(decCreds)),
		option.WithGRPCDialOption(grpc.WithContextDialer(proxy.AwareDialContext)),
	)
}

// CreateSecurityCenterClientFromConfigWithManager creates a client based on the security center config.
func CreateSecurityCenterClientFromConfigWithManager(ctx context.Context,
	manager auth.STSTokenManager, decCreds []byte, wifEnabled bool,
) (*securitycenter.Client, error) {
	if wifEnabled {
		return securitycenter.NewClient(ctx,
			option.WithTokenSource(manager.TokenSource()),
			option.WithGRPCDialOption(grpc.WithContextDialer(proxy.AwareDialContext)),
		)
	}
	return securitycenter.NewClient(ctx,
		option.WithCredentialsJSON([]byte(decCreds)),
		option.WithGRPCDialOption(grpc.WithContextDialer(proxy.AwareDialContext)),
	)
}

// CreateTokenSourceFromConfig creates a token source based on the config.
func CreateTokenSourceFromConfig(ctx context.Context,
	credsJSON []byte, wifEnabled bool, scopes ...string,
) (oauth2.TokenSource, error) {
	if wifEnabled {
		creds, err := google.FindDefaultCredentials(ctx, scopes...)
		if err != nil {
			return nil, err
		}
		return oauth2.ReuseTokenSourceWithExpiry(nil, creds.TokenSource, earlyExpiry), nil
	}
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, scopes...)
	if err != nil {
		return nil, err
	}
	return oauth2.ReuseTokenSourceWithExpiry(nil, creds.TokenSource, earlyExpiry), nil
}

// CreateTokenSourceFromConfigWithManager creates a token source based on the config.
func CreateTokenSourceFromConfigWithManager(ctx context.Context, manager auth.STSTokenManager,
	credsJSON []byte, wifEnabled bool, scopes ...string,
) (oauth2.TokenSource, error) {
	if wifEnabled {
		return manager.TokenSource(), nil
	}
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, scopes...)
	if err != nil {
		return nil, err
	}
	return oauth2.ReuseTokenSourceWithExpiry(nil, creds.TokenSource, earlyExpiry), nil
}
