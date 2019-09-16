package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_1", "anonymous-auth", "false", "true", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_2", "authorization-mode", "AlwaysAllow", "AlwaysAllow", common.NotContains),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_3", "client-ca-file", "", "", common.Set),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_4", "read-only-port", "0", "0", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_5", "streaming-connection-idle-timeout", "0", "0", common.NotMatches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_6", "protect-kernel-defaults", "true", "false", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_7", "make-iptables-util-chains", "true", "true", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_8", "host-override", "", "", common.Unset),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_9", "event-qps", "0", "5", common.Set),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_4_1:2_1_10", "kubelet", "tls-cert-file", "tls-private-key-file"),
		common.PerNodeDeprecatedCheck("CIS_Kubernetes_v1_4_1:2_1_11", "The --cadvisor-port parameter was deprecated in Kubernetes 1.12."),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_12", "rotate-certificates", "false", "true", common.NotMatches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_13", "feature-gates", "RotateKubeletServerCertificate=false", "RotateKubeletServerCertificate=true", common.NotContains),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_4_1:2_1_14", "tls-cipher-suites", tlsCiphers, "", common.OnlyContains),
	)
}

func kubeletCommandLineCheck(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "kubelet", key, target, defaultVal, evalFunc)
}
