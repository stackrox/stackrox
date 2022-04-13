package kubernetes

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/compliance/framework"
	"github.com/stackrox/stackrox/pkg/compliance/msgfmt"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/apiserver/pkg/apis/config/v1"
)

const tlsCiphers = "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256," +
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256," +
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305," +
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384," +
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305," +
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,"

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("1_2_1"):  common.MasterAPIServerCommandLine("anonymous-auth", "false", "true", common.Matches),
		standards.CISKubeCheckName("1_2_2"):  common.MasterAPIServerCommandLine("basic-auth-file", "", "", common.Unset),
		standards.CISKubeCheckName("1_2_3"):  common.MasterAPIServerCommandLine("token-auth-file", "", "", common.Unset),
		standards.CISKubeCheckName("1_2_4"):  common.MasterAPIServerCommandLine("kubelet-https", "true", "true", common.Matches),
		standards.CISKubeCheckName("1_2_5"):  multipleFlagsSetCheck("kube-apiserver", nil, "kubelet-client-certificate", "kubelet-client-key"),
		standards.CISKubeCheckName("1_2_6"):  common.MasterAPIServerCommandLine("kubelet-certificate-authority", "", "", common.Set),
		standards.CISKubeCheckName("1_2_7"):  common.MasterAPIServerCommandLine("authorization-mode", "AlwaysAllow", "AlwaysAllow", common.NotContains),
		standards.CISKubeCheckName("1_2_8"):  common.MasterAPIServerCommandLine("authorization-mode", "Node", "Node", common.Contains),
		standards.CISKubeCheckName("1_2_9"):  common.MasterAPIServerCommandLine("authorization-mode", "RBAC", "RBAC", common.Contains),
		standards.CISKubeCheckName("1_2_10"): common.MasterAPIServerCommandLine("enable-admission-plugins", "EventRateLimit", "", common.Set),
		standards.CISKubeCheckName("1_2_11"): common.MasterAPIServerCommandLine("enable-admission-plugins", "AlwaysAdmit", "AlwaysAdmit", common.NotContains),
		standards.CISKubeCheckName("1_2_12"): common.MasterAPIServerCommandLine("enable-admission-plugins", "AlwaysPullImages", "", common.Contains),
		standards.CISKubeCheckName("1_2_13"): securityContextDenyChecker(),
		standards.CISKubeCheckName("1_2_14"): common.MasterAPIServerCommandLine("disable-admission-plugins", "ServiceAccount", "", common.NotContains),
		standards.CISKubeCheckName("1_2_15"): common.MasterAPIServerCommandLine("disable-admission-plugins", "NamespaceLifecycle", "NamespaceLifecycle", common.NotContains),
		standards.CISKubeCheckName("1_2_16"): common.MasterAPIServerCommandLine("enable-admission-plugins", "PodSecurityPolicy", "", common.Contains),
		standards.CISKubeCheckName("1_2_17"): common.MasterAPIServerCommandLine("enable-admission-plugins", "NodeRestriction", "", common.Set),
		standards.CISKubeCheckName("1_2_18"): common.MasterAPIServerCommandLine("insecure-bind-address", "", "", common.Unset),
		standards.CISKubeCheckName("1_2_19"): common.MasterAPIServerCommandLine("insecure-port", "0", "0", common.Matches),
		standards.CISKubeCheckName("1_2_20"): common.MasterAPIServerCommandLine("secure-port", "0", "6443", common.NotMatches),
		standards.CISKubeCheckName("1_2_21"): common.MasterAPIServerCommandLine("profiling", "false", "true", common.Matches),
		standards.CISKubeCheckName("1_2_22"): common.MasterAPIServerCommandLine("audit-log-path", "", "", common.Set),
		standards.CISKubeCheckName("1_2_23"): common.MasterAPIServerCommandLine("audit-log-maxage", "", "", common.Set),
		standards.CISKubeCheckName("1_2_24"): common.MasterAPIServerCommandLine("audit-log-maxbackup", "", "", common.Set),
		standards.CISKubeCheckName("1_2_25"): common.MasterAPIServerCommandLine("audit-log-maxsize", "", "", common.Set),
		standards.CISKubeCheckName("1_2_26"): common.MasterAPIServerCommandLine("request-timeout", "", "", common.Set),
		standards.CISKubeCheckName("1_2_27"): common.MasterAPIServerCommandLine("service-account-lookup", "true", "true", common.Matches),
		standards.CISKubeCheckName("1_2_28"): common.MasterAPIServerCommandLine("service-account-key-file", "", "", common.Set),
		standards.CISKubeCheckName("1_2_29"): multipleFlagsSetCheck("kube-apiserver", nil, "etcd-certfile", "etcd-keyfile"),
		standards.CISKubeCheckName("1_2_30"): multipleFlagsSetCheck("kube-apiserver", nil, "tls-cert-file", "tls-private-key-file"),
		standards.CISKubeCheckName("1_2_31"): common.MasterAPIServerCommandLine("client-ca-file", "", "", common.Set),
		standards.CISKubeCheckName("1_2_32"): common.MasterAPIServerCommandLine("etcd-cafile", "", "", common.Set),
		standards.CISKubeCheckName("1_2_33"): common.MasterAPIServerCommandLine("encryption-provider-config", "", "", common.Set),
		standards.CISKubeCheckName("1_2_34"): encryptionProvider(),
		standards.CISKubeCheckName("1_2_35"): common.MasterAPIServerCommandLine("tls-cipher-suites", tlsCiphers, "", common.OnlyContains),
	})
}

func encryptionProvider() *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			process, exists := common.GetProcess(complianceData, common.KubeAPIProcessName)
			if !exists {
				return common.NoteListf("Process %q not found on host therefore check is not applicable", common.KubeAPIProcessName)
			}
			arg := common.GetArgForFlag(process.Args, "encryption-provider-config")
			if arg == nil {
				return common.FailListf("encryption-provider-config is not set, which means that aescbc, secretbox or kms is not in use")
			} else if arg.GetFile() == nil {
				return common.FailListf("No file was found encryption-provider-config value of %q", msgfmt.FormatStrings(arg.GetValues()...))
			}

			var config v1.EncryptionConfiguration
			if err := yaml.Unmarshal(arg.GetFile().GetContent(), &config); err != nil {
				return common.FailListf("Could not parse file %q to check for aescbc, secretbox or kms specification due to %v. Please manually check", arg.GetFile().GetPath(), err)
			}
			if config.Kind != "EncryptionConfig" {
				return common.FailListf("Incorrect configuration kind %q in file %q", config.Kind, arg.GetFile().GetPath())
			}
			for _, resource := range config.Resources {
				for _, provider := range resource.Providers {
					if provider.AESCBC != nil {
						return common.PassList("Provider is set as aescbc")
					}
					if provider.Secretbox != nil {
						return common.PassList("Provider is set as secretbox")
					}
					if provider.KMS != nil {
						return common.PassList("Provider is set as kms")
					}
				}
			}
			return common.FailList("Provider is not set as aescbc, secretbox or kms")
		},
		Metadata: &standards.Metadata{
			InterpretationText: "StackRox checks that the Kubernetes API server uses the `aescbc, kms or secretbox` encryption provider",
			TargetKind:         framework.NodeKind,
		},
	}
}

func securityContextDenyChecker() *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			key := "enable-admission-plugins"
			process, exists := common.GetProcess(complianceData, common.KubeAPIProcessName)
			if !exists {
				return common.NoteListf("Process %q not found on host, therefore check is not applicable", common.KubeAPIProcessName)
			}

			values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, key)
			if len(values) == 0 {
				return common.FailListf("%q is unset", key)
			}
			for _, v := range values {
				if v == "PodSecurityPolicy" {
					return common.PassListf("%q is set to %s", key, msgfmt.FormatStrings(v))
				}
			}
			for _, v := range values {
				if v == "SecurityContextDeny" {
					return common.PassListf("%q is set to %s", key, msgfmt.FormatStrings(v))
				}
			}
			return common.FailListf("%q does not contain PodSecurityPolicy or SecurityContextDeny", key)
		},
		Metadata: &standards.Metadata{
			InterpretationText: "StackRox checks that the admission control plugin SecurityContextDeny is set if PodSecurityPolicy is not used",
			TargetKind:         framework.NodeKind,
		},
	}
}
