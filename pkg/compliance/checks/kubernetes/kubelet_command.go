package kubernetes

import (
	"sort"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/set"
	"k8s.io/kubelet/config/v1beta1"
)

var defaultCiphers = []string{
	"TLS_RSA_WITH_AES_256_GCM_SHA384",
	"TLS_RSA_WITH_AES_256_CBC_SHA",
	"TLS_RSA_WITH_AES_128_GCM_SHA256",
	"TLS_RSA_WITH_AES_128_CBC_SHA",
	"TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
}

var tlsCipherSet = set.NewStringSet(
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("4_2_1"):  wrapKubeletCheck(authenticationCheck),
		standards.CISKubeCheckName("4_2_2"):  wrapKubeletCheck(authorizationCheck),
		standards.CISKubeCheckName("4_2_3"):  wrapKubeletCheck(clientCAFile),
		standards.CISKubeCheckName("4_2_4"):  wrapKubeletCheck(readOnlyPort),
		standards.CISKubeCheckName("4_2_5"):  wrapKubeletCheck(streamingConnectionTimeout),
		standards.CISKubeCheckName("4_2_6"):  wrapKubeletCheck(protectKernelDefaults),
		standards.CISKubeCheckName("4_2_7"):  wrapKubeletCheck(makeIPTableUtilChains),
		standards.CISKubeCheckName("4_2_8"):  wrapKubeletCheck(hostnameOverride),
		standards.CISKubeCheckName("4_2_9"):  wrapKubeletCheck(eventQPS),
		standards.CISKubeCheckName("4_2_10"): wrapKubeletCheck(tlsFiles),
		standards.CISKubeCheckName("4_2_11"): wrapKubeletCheck(rotateCertificates),
		standards.CISKubeCheckName("4_2_12"): wrapKubeletCheck(featureGates),
		standards.CISKubeCheckName("4_2_13"): wrapKubeletCheck(tlsCipherSuites),
	})
}

type kubeletCheck func(configuration *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence

func authenticationCheck(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if *config.Authentication.Anonymous.Enabled {
		return common.FailList("Anonymous authentication is set to true")
	}
	return common.PassList("Anonymous authentication is set to false")
}

func authorizationCheck(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if config.Authorization.Mode == v1beta1.KubeletAuthorizationModeAlwaysAllow {
		return common.FailList("Authorization mode is set to AlwaysAllow")
	}
	return common.PassListf("Authorization mode is set to %s", config.Authorization.Mode)
}

func clientCAFile(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if config.Authentication.X509.ClientCAFile == "" {
		return common.FailList("ClientCAFile is unset")
	}
	return common.PassListf("ClientCAFile is set to %s", config.Authentication.X509.ClientCAFile)
}

func readOnlyPort(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if config.ReadOnlyPort != 0 {
		return common.FailListf("ReadOnlyPort is set to %d instead of 0", config.ReadOnlyPort)
	}
	return common.PassList("ReadOnlyPort is set to 0")
}

func streamingConnectionTimeout(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if config.StreamingConnectionIdleTimeout.Seconds() == 0 {
		return common.FailListf("StreamingConnectionIdleTimeout is set to 0")
	}
	return common.PassListf("StreamingConnectionIdleTimeout is set to %s", config.StreamingConnectionIdleTimeout.Duration.String())
}

func protectKernelDefaults(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if !config.ProtectKernelDefaults {
		return common.FailList("ProtectKernelDefaults is set to false when it should be true")
	}
	return common.PassList("ProtectKernelDefaults is set to true")
}

func makeIPTableUtilChains(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if !*config.MakeIPTablesUtilChains {
		return common.FailList("MakeIPTablesUtilChains is set to false")
	}
	return common.PassList("MakeIPTablesUtilChains is set to true")
}

func hostnameOverride(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if config.HostnameOverride != "" {
		return common.FailListf("--hostname-override is set to %s", config.HostnameOverride)
	}
	return common.PassList("--hostname-override is not set")
}

func eventQPS(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if *config.EventRecordQPS == 0 {
		return common.FailList("EventRecordQPS is set to 0")
	}
	return common.PassListf("EventRecordQPS is set to %d", *config.EventRecordQPS)
}

func rotateCertificates(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if !config.RotateCertificates {
		return common.FailList("RotateCertificates is set to false")
	}
	return common.PassList("RotateCertificates is set to true")
}

func tlsFiles(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if config.TLSCertFile == "" && config.TLSPrivateKeyFile == "" {
		return common.FailList("TLSCertFile and TLSPrivateKeyFile are both unset")
	}
	if config.TLSCertFile == "" {
		return common.FailList("TLSCertFile is unset")
	}
	if config.TLSPrivateKeyFile == "" {
		return common.FailList("TLSPrivateKeyFile is unset")
	}
	return common.PassListf("TLSCertFile and TLSPrivateKeyFile are set to %s and %s respectively", config.TLSCertFile, config.TLSPrivateKeyFile)
}

func featureGates(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if val, ok := config.FeatureGates["RotateKubeletServerCertificate"]; !ok {
		return common.FailList("RotateKubeletServiceCertificate feature gate is not set")
	} else if !val {
		return common.FailList("RotateKubeletServiceCertificate is set to false")
	}
	return common.PassList("RotateKubeletServiceCertificate is set to true")
}

func tlsCipherSuites(config *standards.KubeletConfiguration) []*storage.ComplianceResultValue_Evidence {
	if len(config.TLSCipherSuites) == 0 {
		config.TLSCipherSuites = defaultCiphers
	}
	var unexpectedCiphers []string
	for _, cipher := range config.TLSCipherSuites {
		if !tlsCipherSet.Contains(cipher) {
			unexpectedCiphers = append(unexpectedCiphers, cipher)
		}
	}
	if len(unexpectedCiphers) != 0 {
		sort.Strings(unexpectedCiphers)
		return common.FailListf("TLSCipherSuites contains unexpected ciphers: %q", unexpectedCiphers)
	}
	return common.PassListf("TLSCipherSuites contains only %q", config.TLSCipherSuites)
}

func wrapKubeletCheck(fn kubeletCheck) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			if complianceData.KubeletConfiguration == nil {
				return common.FailList("kubelet configuration is empty")
			}
			return fn(complianceData.KubeletConfiguration)
		},
	}
}
