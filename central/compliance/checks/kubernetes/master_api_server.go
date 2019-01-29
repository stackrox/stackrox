package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"gopkg.in/yaml.v2"
	"k8s.io/apiserver/pkg/server/options/encryptionconfig"
)

const kubeAPIProcessName = "kube-apiserver"

func init() {
	framework.MustRegisterChecks(
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_1", "anonymous-auth", "false", "true", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_2", "basic-auth-file", "", "", common.Unset),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_3", "insecure-allow-any-token", "", "", common.Unset),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_4", "kubelet-https", "true", "true", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_5", "insecure-bind-address", "127.0.0.1", "127.0.0.1", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_6", "insecure-port", "0", "8080", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_7", "secure-port", "0", "6443", common.NotMatches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_8", "profiling", "false", "true", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_9", "repair-malformed-updates", "false", "true", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_10", "admission-control", "AlwaysAdmit", "AlwaysAdmit", common.NotContains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_11", "admission-control", "AlwaysPullImages", "AlwaysAdmit", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_12", "admission-control", "DenyEscalatingExec", "AlwaysAdmit", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_13", "admission-control", "SecurityContextDeny", "AlwaysAdmit", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_14", "admission-control", "NamespaceLifecycle", "AlwaysAdmit", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_15", "audit-log-path", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_16", "audit-log-maxage", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_17", "audit-log-maxbackup", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_18", "audit-log-maxsize", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_19", "authorization-mode", "AlwaysAllow", "AlwaysAllow", common.NotContains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_20", "token-auth-file", "", "", common.Unset),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_21", "kubelet-certificate-authority", "", "", common.Set),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_2_0:1_1_22", "kube-apiserver", "kubelet-client-certificate", "kubelet-client-key"),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_23", "service-account-lookup", "true", "false", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_24", "admission-control", "PodSecurityPolicy", "AlwaysAdmit", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_25", "service-account-key-file", "", "", common.Set),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_2_0:1_1_26", "kube-apiserver", "etcd-certfile", "etcd-keyfile"),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_27", "admission-control", "ServiceAccount", "AlwaysAdmit", common.Contains),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_2_0:1_1_28", "kube-apiserver", "tls-cert-file", "tls-private-key-file"),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_29", "client-ca-file", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_30", "etcd-cafile", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_31", "authorization-mode", "Node", "AlwaysAllow", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_32", "admission-control", "NodeRestriction", "AlwaysAllow", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_33", "experimental-encryption-provider-config", "", "", common.Set),
		encryptionProvider(),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_35", "admission-control", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_36", "feature-gates", "AdvancedAuditing=false", "", common.NotContains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_2_0:1_1_37", "request-timeout", "", "", common.Set),
	)
}

func masterAPIServerCommandLine(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, kubeAPIProcessName, key, target, defaultVal, evalFunc)
}

func encryptionProvider() framework.Check {
	return framework.NewCheckFromFunc("CIS_Kubernetes_v1_2_0:1_1_34", framework.NodeKind, nil, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := common.GetProcess(ret, kubeAPIProcessName)
			if !exists {
				framework.NoteNowf(ctx, "Process %q not found on host therefore check is not applicable", kubeAPIProcessName)
			}
			arg := common.GetArgForFlag(process.Args, "experimental-encryption-provider-config")
			if arg == nil {
				framework.FailNowf(ctx, "experimental-encryption-provider-config is not set, which means that aescbc is not in use")
			} else if arg.GetFile() == nil {
				framework.FailNowf(ctx, "No file was found experimental-encryption-provider-config value of %q", arg.Value)
			}

			var config encryptionconfig.EncryptionConfig
			if err := yaml.Unmarshal(arg.GetFile().GetContent(), &config); err != nil {
				framework.FailNowf(ctx, "Could not parse file %q to check for aescbc specification due to %v. Please manually check", arg.GetFile().GetPath(), err)
			}
			if config.Kind != "EncryptionConfig" {
				framework.FailNowf(ctx, "Incorrect configuration kind %q in file %q", config.Kind, arg.GetFile().GetPath())
				return
			}
			for _, resource := range config.Resources {
				for _, provider := range resource.Providers {

					if provider.AESCBC != nil {
						framework.PassNow(ctx, "Provider is set as aescbc")
					}
				}
			}
			framework.Fail(ctx, "Provider is not set as aescbc")
		}))
}
