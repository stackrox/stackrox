package kubelet

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type kubeconfigFilePermissions struct{}

func (c *kubeconfigFilePermissions) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 2.2.5",
			Description: "Ensure that the proxy kubeconfig file permissions are set to 644 or more restrictive",
		}, Dependencies: []utils.Dependency{utils.InitKubeProxyConfig},
	}
}

func (c *kubeconfigFilePermissions) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	params, ok := utils.KubeletConfig.Get("kubeconfig")
	if !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "Cannot check kubeconfig file permissions because kube-proxy command line does not define 'kubeconfig' parameter")
		return
	}

	result = utils.NewPermissionsCheck("", "", params.String(), 0644, true).Run()
	return
}

// NewKubeconfigFilePermissions implements CIS Kubernetes v1.2.0 2.2.5
func NewKubeconfigFilePermissions() utils.Check {
	return &kubeconfigFilePermissions{}
}

func init() {
	checks.AddToRegistry(
		NewKubeconfigFilePermissions(),
	)
}
