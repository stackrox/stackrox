package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/msgfmt"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/apiserver/pkg/apis/config/v1"
)

const kubeAPIProcessName = "kube-apiserver"

const tlsCiphers = "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256," +
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256," +
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305," +
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384," +
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305," +
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,"

func init() {
	framework.MustRegisterChecks(
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_1", "anonymous-auth", "false", "true", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_2", "basic-auth-file", "", "", common.Unset),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_3", "token-auth-file", "", "", common.Unset),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_4", "kubelet-https", "true", "true", common.Matches),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_5:1_2_5", "kube-apiserver", "kubelet-client-certificate", "kubelet-client-key"),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_6", "kubelet-certificate-authority", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_7", "authorization-mode", "AlwaysAllow", "AlwaysAllow", common.NotContains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_8", "authorization-mode", "Node", "Node", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_9", "authorization-mode", "RBAC", "RBAC", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_10", "enable-admission-plugins", "EventRateLimit", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_11", "enable-admission-plugins", "AlwaysAdmit", "AlwaysAdmit", common.NotContains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_12", "enable-admission-plugins", "AlwaysPullImages", "", common.Contains),
		securityContextDenyChecker(),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_14", "disable-admission-plugins", "ServiceAccount", "", common.NotContains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_15", "disable-admission-plugins", "NamespaceLifecycle", "NamespaceLifecycle", common.NotContains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_16", "enable-admission-plugins", "PodSecurityPolicy", "", common.Contains),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_17", "enable-admission-plugins", "NodeRestriction", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_18", "insecure-bind-address", "", "", common.Unset),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_19", "insecure-port", "0", "0", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_20", "secure-port", "0", "6443", common.NotMatches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_21", "profiling", "false", "true", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_22", "audit-log-path", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_23", "audit-log-maxage", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_24", "audit-log-maxbackup", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_25", "audit-log-maxsize", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_26", "request-timeout", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_27", "service-account-lookup", "true", "true", common.Matches),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_28", "service-account-key-file", "", "", common.Set),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_5:1_2_29", "kube-apiserver", "etcd-certfile", "etcd-keyfile"),
		multipleFlagsSetCheck("CIS_Kubernetes_v1_5:1_2_30", "kube-apiserver", "tls-cert-file", "tls-private-key-file"),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_31", "client-ca-file", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_32", "etcd-cafile", "", "", common.Set),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_33", "encryption-provider-config", "", "", common.Set),
		encryptionProvider(),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:1_2_35", "tls-cipher-suites", tlsCiphers, "", common.OnlyContains),
	)
}

func masterAPIServerCommandLine(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, kubeAPIProcessName, key, target, defaultVal, evalFunc)
}

func encryptionProvider() framework.Check {
	md := framework.CheckMetadata{
		ID:                 "CIS_Kubernetes_v1_5:1_2_34",
		Scope:              framework.NodeKind,
		InterpretationText: "StackRox checks that the Kubernetes API server uses the `aescbc, kms or secretbox` encryption provider",
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := common.GetProcess(ret, kubeAPIProcessName)
			if !exists {
				framework.NoteNowf(ctx, "Process %q not found on host therefore check is not applicable", kubeAPIProcessName)
			}
			arg := common.GetArgForFlag(process.Args, "encryption-provider-config")
			if arg == nil {
				framework.FailNowf(ctx, "encryption-provider-config is not set, which means that aescbc, secretbox or kms is not in use")
			} else if arg.GetFile() == nil {
				framework.FailNowf(ctx, "No file was found encryption-provider-config value of %q", msgfmt.FormatStrings(arg.GetValues()...))
			}

			var config v1.EncryptionConfiguration
			if err := yaml.Unmarshal(arg.GetFile().GetContent(), &config); err != nil {
				framework.FailNowf(ctx, "Could not parse file %q to check for aescbc, secretbox or kms specification due to %v. Please manually check", arg.GetFile().GetPath(), err)
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
					if provider.Secretbox != nil {
						framework.PassNow(ctx, "Provider is set as secretbox")
					}
					if provider.KMS != nil {
						framework.PassNow(ctx, "Provider is set as kms")
					}
				}
			}
			framework.Fail(ctx, "Provider is not set as aescbc, secretbox or kms")
		}))
}

func securityContextDenyChecker() framework.Check {
	md := framework.CheckMetadata{
		ID:                 "CIS_Kubernetes_v1_5:1_2_13",
		Scope:              framework.NodeKind,
		InterpretationText: "StackRox checks that the admission control plugin SecurityContextDeny is set if PodSecurityPolicy is not used",
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			key := "enable-admission-plugins"
			process, exists := common.GetProcess(ret, kubeAPIProcessName)
			if !exists {
				framework.NoteNowf(ctx, "Process %q not found on host, therefore check is not applicable", kubeAPIProcessName)
			}

			values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, key)
			if len(values) == 0 {
				framework.Failf(ctx, "%q is unset", key)
			} else {
				for _, v := range values {
					if v == "PodSecurityPolicy" {
						framework.PassNowf(ctx, "%q is set to %s", key, msgfmt.FormatStrings(v))
					}
				}
				for _, v := range values {
					if v == "SecurityContextDeny" {
						framework.PassNowf(ctx, "%q is set to %s", key, msgfmt.FormatStrings(v))
					}
				}
				framework.Failf(ctx, "%q does not contain PodSecurityPolicy or SecurityContextDeny", key)
			}
		}))
}
