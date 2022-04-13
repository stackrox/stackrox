package sensor

import "github.com/stackrox/stackrox/pkg/env"

var (
	k8sNodeName = env.RegisterSetting("K8S_NODE_NAME")
)
