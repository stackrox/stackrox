package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_1", "anonymous-auth", "false", "true", common.Matches),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_2", "basic-auth-file", "", "", common.Unset),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_3", "insecure-allow-any-token", "", "", common.Unset),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_4", "insecure-bind-address", "", "", common.Unset),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_5", "insecure-port", "0", "8080", common.Matches),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_6", "secure-port", "0", "6443", common.NotMatches),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_7", "profiling", "false", "true", common.Matches),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_8", "admission-control", "AlwaysAdmit", "AlwaysAdmit", common.NotContains),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_9", "admission-control", "NamespaceLifecycle", "AlwaysAdmit", common.Contains),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_10", "audit-log-path", "", "", common.Set),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_11", "audit-log-maxage", "", "", common.Set),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_12", "audit-log-maxbackup", "", "", common.Set),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_13", "audit-log-maxsize", "", "", common.Set),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_14", "authorization-mode", "AlwaysAllow", "AlwaysAllow", common.NotContains),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_15", "token-auth-file", "", "", common.Unset),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_16", "service-account-lookup", "true", "false", common.Matches),
		federatedAPIServerCommandLine("CIS_Kubernetes_v1_2_0:3_1_17", "service-account-key-file", "", "", common.Set),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_2_0:3_1_18", "etcd-certfile", "etcd-keyfile"),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_2_0:3_1_19", "tls-cert-file", "tls-private-key"),
	)
}

func federatedAPIServerCommandLine(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "federation-apiserver", key, target, defaultVal, evalFunc)
}
