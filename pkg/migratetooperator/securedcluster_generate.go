package migratetooperator

import (
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TransformToSecuredCluster detects the configuration from the given source and
// generates a SecuredCluster custom resource. It returns the CR and a list of
// warnings for the caller to emit.
func TransformToSecuredCluster(src Source) (*platform.SecuredCluster, []string, error) {
	config, err := detectSecuredCluster(src)
	if err != nil {
		return nil, nil, errors.Wrap(err, "detecting secured cluster configuration")
	}
	cr, warnings := generateSecuredCluster(config)
	return cr, warnings, nil
}

func generateSecuredCluster(config *securedClusterConfig) (*platform.SecuredCluster, []string) {
	var warnings []string

	cr := &platform.SecuredCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "SecuredCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "stackrox-secured-cluster-services",
		},
		Spec: platform.SecuredClusterSpec{
			ClusterName: pointers.String(config.clusterName),
		},
	}

	if config.centralEndpoint != "" && config.centralEndpoint != "central.stackrox:443" {
		cr.Spec.CentralEndpoint = pointers.String(config.centralEndpoint)
	}

	if config.enforcementDisabled || config.failurePolicyFail {
		ac := &platform.AdmissionControlComponentSpec{}
		if config.enforcementDisabled {
			ac.Enforcement = (*platform.PolicyEnforcement)(pointers.String(string(platform.PolicyEnforcementDisabled)))
		}
		if config.failurePolicyFail {
			ac.FailurePolicy = (*platform.FailurePolicy)(pointers.String(string(platform.FailurePolicyFail)))
		}
		cr.Spec.AdmissionControl = ac
	}

	if config.collectionNone || config.tolerationsDisabled {
		perNode := &platform.PerNodeSpec{}
		if config.collectionNone {
			perNode.Collector = &platform.CollectorContainerSpec{
				Collection: (*platform.CollectionMethod)(pointers.String(string(platform.CollectionNone))),
			}
		}
		if config.tolerationsDisabled {
			perNode.TaintToleration = (*platform.TaintTolerationPolicy)(pointers.String(string(platform.TaintAvoid)))
		}
		cr.Spec.PerNode = perNode
	}

	if config.customImages {
		warnings = append(warnings, "Detected non-default container images. "+
			"The operator does not support image overrides in the SecuredCluster CR. "+
			"Configure RELATED_IMAGE_* environment variables on the operator Deployment instead.")
	}

	// TODO: The following options are stored as server-side cluster configuration
	// in Central and are not reflected in the generated sensor manifests:
	//   - --admission-controller-disable-bypass → spec.admissionControl.bypass
	//   - --auto-lock-process-baselines → spec.processBaselines.autoLock
	//   - --disable-audit-logs → spec.auditLogs.collection
	// To detect these, the tool would need to query the Central API
	// (e.g. GET /v1/clusters/<id>) to read the cluster's runtime configuration.
	// This could be implemented for the --namespace (live cluster) mode by
	// reading the cluster config from the API using the same credentials.

	return cr, warnings
}
