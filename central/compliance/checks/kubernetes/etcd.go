package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		multipleFlagsSetCheck("CIS_Kubernetes_v1_4_1:1_5_1", "etcd", "cert-file", "key-file"),
		etcdCommandLineCheck("CIS_Kubernetes_v1_4_1:1_5_2", "client-cert-auth", "true", "false", common.Matches),
		etcdCommandLineCheck("CIS_Kubernetes_v1_4_1:1_5_3", "auto-tls", "true", "false", common.NotMatches),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_4_1:1_5_4", "etcd", "peer-cert-file", "peer-key-file"),
		etcdCommandLineCheck("CIS_Kubernetes_v1_4_1:1_5_5", "peer-client-cert-auth", "true", "false", common.Matches),
		etcdCommandLineCheck("CIS_Kubernetes_v1_4_1:1_5_6", "peer-auto-tls", "true", "false", common.NotMatches),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_4_1:1_5_7", "Ensure that a unique Certificate Authority is used for etcd"),
	)
}

func etcdCommandLineCheck(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "etcd", key, target, defaultVal, evalFunc)
}
