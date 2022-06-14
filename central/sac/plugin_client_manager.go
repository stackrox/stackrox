package sac

import (
	"github.com/stackrox/stackrox/pkg/sac/client"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	managerInstance     AuthPluginClientManger
	managerInstanceInit sync.Once
)

// AuthPluginClientManger implementations must provide access to the auth plugin client if it has been configured.
//go:generate mockgen-wrapper
type AuthPluginClientManger interface {
	SetClient(newClient client.Client)
	GetClient() client.Client
}

// AuthPluginClientManagerSingleton returns the singleton instance of the deployment environments manager.
func AuthPluginClientManagerSingleton() AuthPluginClientManger {
	managerInstanceInit.Do(func() {
		managerInstance = &authPluginClientManager{}
	})
	return managerInstance
}
