package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		multipleFlagsSetCheck("CIS_Kubernetes_v1_2_0:1_5_1", "etcd", "cert-file", "key-file"),
		etcdCommandLineCheck("CIS_Kubernetes_v1_2_0:1_5_2", "client-cert-auth", "true", "false", common.Matches),
		etcdCommandLineCheck("CIS_Kubernetes_v1_2_0:1_5_3", "auto-tls", "true", "false", common.Matches),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_2_0:1_5_4", "etcd", "peer-cert-file", "peer-key-file"),
		etcdCommandLineCheck("CIS_Kubernetes_v1_2_0:1_5_5", "peer-client-cert-auth", "true", "false", common.Matches),
		etcdCommandLineCheck("CIS_Kubernetes_v1_2_0:1_5_6", "peer-auto-tls", "true", "false", common.Matches),
		etcdCommandLineCheck("CIS_Kubernetes_v1_2_0:1_5_7", "wal-dir", "", "", common.Set),
		etcdCommandLineCheck("CIS_Kubernetes_v1_2_0:1_5_8", "max-wals", "0", "5", common.Matches),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_5_9", "Ensure that a unique Certificate Authority is used for etcd"),
	)
}

func etcdCommandLineCheck(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "etcd", key, target, defaultVal, evalFunc)
}
