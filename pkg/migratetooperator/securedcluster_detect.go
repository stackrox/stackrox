package migratetooperator

import (
	"strings"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
)

type securedClusterConfig struct {
	clusterName         string
	centralEndpoint     string
	enforcementDisabled bool
	failurePolicyFail   bool
	collectionNone      bool
	tolerationsDisabled bool
	customImages        bool
}

func detectSecuredCluster(src Source) (*securedClusterConfig, error) {
	clusterName, err := detectClusterName(src)
	if err != nil {
		return nil, errors.Wrap(err, "detecting cluster name")
	}

	sensorDep, err := src.Deployment("sensor")
	if err != nil {
		return nil, errors.Wrap(err, "retrieving sensor Deployment")
	}

	centralEndpoint := envVarValue(sensorDep, "ROX_CENTRAL_ENDPOINT")

	vwc, err := src.ValidatingWebhookConfiguration("stackrox")
	if err != nil {
		return nil, errors.Wrap(err, "checking for admission controller webhooks")
	}
	enforcementDisabled := !hasWebhook(vwc, "policyeval.stackrox.io")
	failurePolicyFail := hasFailurePolicyFail(vwc)

	collectorDS, err := src.DaemonSet("collector")
	if err != nil {
		return nil, errors.Wrap(err, "retrieving collector DaemonSet")
	}
	collectionNone := !hasContainer(collectorDS, "collector")
	tolerationsDisabled := len(collectorDS.Spec.Template.Spec.Tolerations) == 0

	return &securedClusterConfig{
		clusterName:         clusterName,
		centralEndpoint:     centralEndpoint,
		enforcementDisabled: enforcementDisabled,
		failurePolicyFail:   failurePolicyFail,
		collectionNone:      collectionNone,
		tolerationsDisabled: tolerationsDisabled,
		customImages:        detectCustomImages(sensorDep),
	}, nil
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
