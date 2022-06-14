package sac

import (
	"github.com/stackrox/rox/pkg/sac/client"
	"github.com/stackrox/rox/pkg/sync"
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
