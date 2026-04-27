package migratetooperator

import (
	"strings"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
)

// TransformToSecuredCluster detects the configuration from the given source and
// generates a SecuredCluster custom resource. It returns the CR and a list of
// warnings for the caller to emit.
func TransformToSecuredCluster(src Source) (*platform.SecuredCluster, []string, error) {
	var warnings []string

	// Cluster name
	clusterName, err := detectClusterName(src)
	if err != nil {
		return nil, nil, errors.Wrap(err, "detecting cluster name")
	}

	cr := &platform.SecuredCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "SecuredCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "stackrox-secured-cluster-services",
		},
		Spec: platform.SecuredClusterSpec{
			ClusterName: pointers.String(clusterName),
		},
	}

	// Central endpoint
	sensorDep, err := src.Deployment("sensor")
	if err != nil {
		return nil, nil, errors.Wrap(err, "retrieving sensor Deployment")
	}
	if sensorDep == nil {
		return nil, nil, errors.New("sensor Deployment not found")
	}
	if ep := envVarValue(sensorDep, "ROX_CENTRAL_ENDPOINT"); ep != "" && ep != "central.stackrox:443" {
		cr.Spec.CentralEndpoint = pointers.String(ep)
	}

	// Admission controller
	vwc, err := src.ValidatingWebhookConfiguration("stackrox")
	if err != nil {
		return nil, nil, errors.Wrap(err, "checking for admission controller webhooks")
	}
	if vwc != nil {
		ac := &platform.AdmissionControlComponentSpec{}
		changed := false
		if !hasWebhook(vwc, "policyeval.stackrox.io") {
			enforcement := platform.PolicyEnforcementDisabled
			ac.Enforcement = &enforcement
			changed = true
		}
		if hasFailurePolicyFail(vwc) {
			fp := platform.FailurePolicyFail
			ac.FailurePolicy = &fp
			changed = true
		}
		if changed {
			cr.Spec.AdmissionControl = ac
		}
	}

	// Collector
	collectorDS, err := src.DaemonSet("collector")
	if err != nil {
		return nil, nil, errors.Wrap(err, "retrieving collector DaemonSet")
	}
	if collectorDS == nil {
		return nil, nil, errors.New("collector DaemonSet not found")
	}
	collectionNone := !hasContainer(collectorDS, "collector")
	tolerationsDisabled := len(collectorDS.Spec.Template.Spec.Tolerations) == 0
	if collectionNone || tolerationsDisabled {
		perNode := &platform.PerNodeSpec{}
		if collectionNone {
			cm := platform.CollectionNone
			perNode.Collector = &platform.CollectorContainerSpec{
				Collection: &cm,
			}
		}
		if tolerationsDisabled {
			tt := platform.TaintAvoid
			perNode.TaintToleration = &tt
		}
		cr.Spec.PerNode = perNode
	}

	// Custom images
	if detectCustomImages(sensorDep) {
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

	return cr, warnings, nil
}

func detectClusterName(src Source) (string, error) {
	secret, err := src.Secret("helm-effective-cluster-name")
	if err != nil {
		return "", errors.Wrap(err, "looking up helm-effective-cluster-name Secret")
	}
	if secret == nil {
		return "", errors.New("Secret \"helm-effective-cluster-name\" not found")
	}
	name := secret.StringData["cluster-name"]
	if name == "" {
		if raw, ok := secret.Data["cluster-name"]; ok {
			name = string(raw)
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("cluster name is empty in Secret \"helm-effective-cluster-name\"")
	}
	return name, nil
}

func hasWebhook(vwc *admissionv1.ValidatingWebhookConfiguration, name string) bool {
	if vwc == nil {
		return false
	}
	for _, wh := range vwc.Webhooks {
		if wh.Name == name {
			return true
		}
	}
	return false
}

func hasFailurePolicyFail(vwc *admissionv1.ValidatingWebhookConfiguration) bool {
	if vwc == nil {
		return false
	}
	for _, wh := range vwc.Webhooks {
		if wh.FailurePolicy != nil && *wh.FailurePolicy == admissionv1.Fail {
			return true
		}
	}
	return false
}

func hasContainer(ds *appsv1.DaemonSet, name string) bool {
	for _, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == name {
			return true
		}
	}
	return false
}
