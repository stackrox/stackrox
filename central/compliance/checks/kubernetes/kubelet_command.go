package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_1", "allow-privileged", "false", "true", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_2", "anonymous-auth", "false", "true", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_3", "authorization-mode", "AlwaysAllow", "AlwaysAllow", common.NotContains),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_4", "client-ca-file", "", "", common.Set),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_5", "read-only-port", "0", "0", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_6", "streaming-connection-idle-timeout", "0", "0", common.NotMatches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_7", "protect-kernel-defaults", "true", "false", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_8", "make-iptables-util-chains", "true", "true", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_9", "keep-terminated-pod-volumes", "false", "true", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_10", "host-override", "", "", common.Unset),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_11", "event-qps", "0", "5", common.Set),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_2_0:2_1_12", "kubelet", "tls-cert-file", "tls-private-key-file"),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_13", "cadvisor-port", "0", "0", common.Matches),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_14", "feature-gates", "RotateKubeletClientCertificate=false", "RotateKubeletClientCertificate=true", common.NotContains),
		kubeletCommandLineCheck("CIS_Kubernetes_v1_2_0:2_1_15", "feature-gates", "RotateKubeletServerCertificate=false", "RotateKubeletServerCertificate=true", common.NotContains),
	)
}

func kubeletCommandLineCheck(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "kubelet", key, target, defaultVal, evalFunc)
}
