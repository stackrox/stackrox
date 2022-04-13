package kubernetes

import (
	"strings"

	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/compliance/framework"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("1_1_1"): common.OptionalPermissionCheck("/etc/kubernetes/manifests/kube-apiserver.yaml", 0644),
		standards.CISKubeCheckName("1_1_2"): common.OptionalOwnershipCheck("/etc/kubernetes/manifests/kube-apiserver.yaml", "root", "root"),

		standards.CISKubeCheckName("1_1_3"): common.OptionalPermissionCheck("/etc/kubernetes/manifests/kube-controller-manager.yaml", 0644),
		standards.CISKubeCheckName("1_1_4"): common.OptionalOwnershipCheck("/etc/kubernetes/manifests/kube-controller-manager.yaml", "root", "root"),

		standards.CISKubeCheckName("1_1_5"): common.OptionalPermissionCheck("/etc/kubernetes/manifests/kube-scheduler.yaml", 0644),
		standards.CISKubeCheckName("1_1_6"): common.OptionalOwnershipCheck("/etc/kubernetes/manifests/kube-scheduler.yaml", "root", "root"),

		standards.CISKubeCheckName("1_1_7"): common.OptionalPermissionCheck("/etc/kubernetes/manifests/etcd.yaml", 0644),
		standards.CISKubeCheckName("1_1_8"): common.OptionalOwnershipCheck("/etc/kubernetes/manifests/etcd.yaml", "root", "root"),

		standards.CISKubeCheckName("1_1_9"):  cniFilePermissions(),
		standards.CISKubeCheckName("1_1_10"): cniFileOwnership(),

		standards.CISKubeCheckName("1_1_11"): etcdDataPermissions(),
		standards.CISKubeCheckName("1_1_12"): etcdDataOwnership(),

		standards.CISKubeCheckName("1_1_13"): common.OptionalPermissionCheck("/etc/kubernetes/manifests/admin.conf", 0644),
		standards.CISKubeCheckName("1_1_14"): common.OptionalOwnershipCheck("/etc/kubernetes/manifests/admin.conf", "root", "root"),

		standards.CISKubeCheckName("1_1_15"): common.OptionalPermissionCheck("/etc/kubernetes/scheduler.conf", 0644),
		standards.CISKubeCheckName("1_1_16"): common.OptionalOwnershipCheck("/etc/kubernetes/scheduler.conf", "root", "root"),

		standards.CISKubeCheckName("1_1_17"): common.OptionalPermissionCheck("/etc/kubernetes/controller-manager.conf", 0644),
		standards.CISKubeCheckName("1_1_18"): common.OptionalOwnershipCheck("/etc/kubernetes/controller-manager.conf", "root", "root"),

		standards.CISKubeCheckName("1_1_19"): common.RecursiveOwnershipCheckIfDirExists("/etc/kubernetes/pki", "root", "root"),
		standards.CISKubeCheckName("1_1_20"): common.RecursivePermissionCheckWithFileExtIfDirExists("/etc/kubernetes/pki", ".crt", 0644),
		standards.CISKubeCheckName("1_1_21"): common.RecursivePermissionCheckWithFileExtIfDirExists("/etc/kubernetes/pki", ".key", 0600),
	})
}

func getDirectoryFileFromCommandLine(complianceData *standards.ComplianceData, processName string, flag, defaultVal string) (*compliance.File, *storage.ComplianceResultValue_Evidence) {
	process, exists := common.GetProcess(complianceData, processName)
	if !exists {
		return nil, common.Notef("Process %q not found on host therefore check is not applicable", processName)
	}
	var dir string
	values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, flag)
	if len(values) == 0 {
		dir = defaultVal
	} else {
		dir = values[0]
	}
	dir = strings.TrimRight(dir, "/")
	dirFile, exists := complianceData.Files[dir]
	if !exists {
		return nil, common.Failf("%q directory does not exist", dir)
	}
	return dirFile, nil
}

func cniFilePermissions() *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			var results []*storage.ComplianceResultValue_Evidence
			dirFile, result := getDirectoryFileFromCommandLine(complianceData, "kubelet", "cni-conf-dir", "/etc/cni/net.d")
			if result != nil {
				results = append(results, result)
			}
			if dirFile != nil {
				permissionResults, failNow := common.CheckRecursivePermissions(dirFile, 0644)
				results = append(results, permissionResults...)
				if failNow {
					return results
				}
			}

			dirFile, result = getDirectoryFileFromCommandLine(complianceData, "kubelet", "cni-bin-dir", "/opt/cni/bin")
			if result != nil {
				results = append(results, result)
			}
			if dirFile != nil {
				permissionResults, failNow := common.CheckRecursivePermissions(dirFile, 0644)
				results = append(results, permissionResults...)
				if failNow {
					return results
				}
			}
			return results
		},
		Metadata: &standards.Metadata{
			InterpretationText: "StackRox checks that the permissions of files in the CNI configuration and binary directories are set to at most '0644'",
			TargetKind:         framework.NodeKind,
		},
	}
}

func cniFileOwnership() *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			var results []*storage.ComplianceResultValue_Evidence
			dirFile, result := getDirectoryFileFromCommandLine(complianceData, "kubelet", "cni-conf-dir", "/etc/cni/net.d")
			if result != nil {
				results = append(results, result)
			}
			if dirFile != nil {
				results = append(results, common.CheckRecursiveOwnership(dirFile, "root", "root")...)
			}
			dirFile, result = getDirectoryFileFromCommandLine(complianceData, "kubelet", "cni-bin-dir", "/opt/cni/bin")
			if result != nil {
				results = append(results, result)
			}
			if dirFile != nil {
				results = append(results, common.CheckRecursiveOwnership(dirFile, "root", "root")...)
			}
			return results
		},
		Metadata: &standards.Metadata{
			InterpretationText: "StackRox checks that the owner and group of files in the CNI configuration and binary directories is root:root",
			TargetKind:         framework.NodeKind,
		},
	}
}

func etcdDataPermissions() *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			var results []*storage.ComplianceResultValue_Evidence
			dirFile, result := getDirectoryFileFromCommandLine(complianceData, "etcd", "data-dir", "/var/lib/etcddisk")
			if result != nil {
				results = append(results, result)
			}
			if dirFile != nil {
				permissionResults, failNow := common.CheckRecursivePermissions(dirFile, 0700)
				results = append(results, permissionResults...)
				if failNow {
					return results
				}
			}
			return results
		},
		Metadata: &standards.Metadata{
			InterpretationText: "StackRox checks that the permissions of the etcd data directory are set to '0700'",
			TargetKind:         framework.NodeKind,
		},
	}
}

func etcdDataOwnership() *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			var results []*storage.ComplianceResultValue_Evidence
			dirFile, result := getDirectoryFileFromCommandLine(complianceData, "etcd", "data-dir", "/var/lib/etcddisk")
			if result != nil {
				results = append(results, result)
			}
			if dirFile != nil {
				results = append(results, common.CheckRecursiveOwnership(dirFile, "etcd", "etcd")...)
			}
			return results
		},
		Metadata: &standards.Metadata{
			InterpretationText: "StackRox checks that the owner and group of the etcd data directory are set to etcd:etcd",
			TargetKind:         framework.NodeKind,
		},
	}
}
