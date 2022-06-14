package sac

import (
	"github.com/stackrox/rox/pkg/sac/client"
	"github.com/stackrox/rox/pkg/sync"
)

type authPluginClientManager struct {
	lock             sync.Mutex
	authPluginClient client.Client
}

func (pcm *authPluginClientManager) SetClient(newClient client.Client) {
	pcm.lock.Lock()
	defer pcm.lock.Unlock()

	pcm.authPluginClient = newClient
}

func (pcm *authPluginClientManager) GetClient() client.Client {
	pcm.lock.Lock()
	defer pcm.lock.Unlock()

	return pcm.authPluginClient
}
