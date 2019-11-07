package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_1", "anonymous-auth", "false", "true", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_2", "authorization-mode", "AlwaysAllow", "AlwaysAllow", common.NotContains),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_3", "client-ca-file", "", "", common.Set),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_4", "read-only-port", "0", "10255/TCP", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_5", "streaming-connection-idle-timeout", "0", "0", common.NotMatches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_6", "protect-kernel-defaults", "true", "false", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_7", "make-iptables-util-chains", "true", "true", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_8", "host-override", "", "", common.Unset),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_9", "event-qps", "0", "5", common.Set),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_5:4_2_10", "kubelet", "tls-cert-file", "tls-private-key-file"),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_11", "rotate-certificates", "false", "true", common.NotMatches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_12", "feature-gates", "RotateKubeletServerCertificate=true", "RotateKubeletServerCertificate=false", common.Contains),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_5:4_2_13", "tls-cipher-suites", tlsCiphers, "", common.OnlyContains),
	)
}

func kubeletCommandLineCheck(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "kubelet", key, target, defaultVal, evalFunc)
}
