package kubernetes

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("2_1"): multipleFlagsSetCheck("etcd", nil, "cert-file", "key-file"),
		standards.CISKubeCheckName("2_2"): etcdCommandLineCheck("client-cert-auth", "true", "false", common.Matches),
		standards.CISKubeCheckName("2_3"): etcdCommandLineCheck("auto-tls", "true", "false", common.NotMatches),
		standards.CISKubeCheckName("2_4"): multipleFlagsSetCheck("etcd", nil, "peer-cert-file", "peer-key-file"),
		standards.CISKubeCheckName("2_5"): etcdCommandLineCheck("peer-client-cert-auth", "true", "false", common.Matches),
		standards.CISKubeCheckName("2_6"): etcdCommandLineCheck("peer-auto-tls", "true", "false", common.NotMatches),
		standards.CISKubeCheckName("2_7"): common.NoteCheck("Ensure that a unique Certificate Authority is used for etcd"),
	})
}

func etcdCommandLineCheck(key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) *standards.CheckAndMetadata {
	return genericKubernetesCommandlineCheck("etcd", key, target, defaultVal, evalFunc)
}
