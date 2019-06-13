package sac

import "github.com/stackrox/rox/pkg/sac/client"

type authPluginClientManager struct {
}

func (pcm *authPluginClientManager) GetClient() client.Client {
	return nil
}
