package kubelet

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

/*
// NewKubeletServicePermission implements CIS Kubernetes v1.2.0 2.2.3
func NewKubeletServicePermission() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 2.2.3",
		"Ensure that the kubelet service file permissions are set to 644 or more restrictive",
		"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf",
		0644,
		true,
	)
}
*/
type kubeletSystemdPermissions struct{}

func (c *kubeletSystemdPermissions) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 2.2.3",
			Description: "Ensure that the kubelet service file permissions are set to 644 or more restrictive",
		},
	}
}

func (c *kubeletSystemdPermissions) Run() (result v1.BenchmarkCheckResult) {
	utils.Pass(&result)

	systemdPath, err := utils.GetSystemdFile("kubelet.service")
	if err != nil {
		utils.Note(&result)
		utils.AddNotef(&result, "Systemd path for service %v is not found. Test may not be applicable", systemdPath)
		return
	}

	result = utils.NewPermissionsCheck("", "", systemdPath, 0644, true).Run()

	res := utils.NewRecursivePermissionsCheck("", "", systemdPath+".d", 0644, true).Run()
	if result.Result != v1.BenchmarkCheckStatus_PASS {
		result.Result = res.Result
	}
	utils.AddNotes(&result, res.Notes...)
	return
}

// NewKubeletSystemdPermissions implements CIS Kubernetes v1.2.0 2.2.3
func NewKubeletSystemdPermissions() utils.Check {
	return &kubeletSystemdPermissions{}
}

func init() {
	checks.AddToRegistry(
		NewKubeletSystemdPermissions(),
	)
}
