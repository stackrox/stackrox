package kubernetes

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("4_2_1"):  kubeletCommandLineCheck("anonymous-auth", "false", "true", common.Matches),
		standards.CISKubeCheckName("4_2_2"):  kubeletCommandLineCheck("authorization-mode", "AlwaysAllow", "AlwaysAllow", common.NotContains),
		standards.CISKubeCheckName("4_2_3"):  kubeletCommandLineCheck("client-ca-file", "", "", common.Set),
		standards.CISKubeCheckName("4_2_4"):  kubeletCommandLineCheck("read-only-port", "0", "10255/TCP", common.Matches),
		standards.CISKubeCheckName("4_2_5"):  kubeletCommandLineCheck("streaming-connection-idle-timeout", "0", "4h", common.NotMatches),
		standards.CISKubeCheckName("4_2_6"):  kubeletCommandLineCheck("protect-kernel-defaults", "true", "false", common.Matches),
		standards.CISKubeCheckName("4_2_7"):  kubeletCommandLineCheck("make-iptables-util-chains", "true", "true", common.Matches),
		standards.CISKubeCheckName("4_2_8"):  kubeletCommandLineCheck("host-override", "", "", common.Unset),
		standards.CISKubeCheckName("4_2_9"):  kubeletCommandLineCheck("event-qps", "0", "5", common.Info),
		standards.CISKubeCheckName("4_2_10"): multipleFlagsSetCheck("kubelet", kubeletCommandLineOverride, "tls-cert-file", "tls-private-key-file"),
		standards.CISKubeCheckName("4_2_11"): kubeletCommandLineCheck("rotate-certificates", "false", "true", common.NotMatches),
		standards.CISKubeCheckName("4_2_12"): kubeletCommandLineCheck("feature-gates", "RotateKubeletServerCertificate=true", "RotateKubeletServerCertificate=false", common.Contains),
		standards.CISKubeCheckName("4_2_13"): kubeletCommandLineCheck("tls-cipher-suites", tlsCiphers, defaultTLSCiphers, common.OnlyContains),
	})
}

func kubeletCommandLineOverride(msg string) []*storage.ComplianceResultValue_Evidence {
	return common.NoteListf("%s. Please check the kubelet config file to verify this result.", msg)
}

func kubeletCommandLineCheck(key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) *standards.CheckAndMetadata {
	return genericKubernetesCommandlineCheck("kubelet", key, target, defaultVal, evalFunc, kubeletCommandLineOverride)
}
