package complianceoperator

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	appsv1 "k8s.io/api/apps/v1"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

var ErrUnableToExtractVersion = errors.New("compliance operator found " +
	"but labels required for extracting the version are missing")

func GetInstalledVersion(ctx context.Context, ns string, dynClient dynamic.Interface) (ver, namespace string, err error) {
	complianceOperatorDeployment, err := searchForDeployment(ctx, ns, dynClient)
	if err != nil {
		return "", ns, errors.Wrapf(err, "could not find compliance operator deployment %q", ns)
	}

	foundInNamespace := complianceOperatorDeployment.GetNamespace()
	version := extractVersionFromLabels(complianceOperatorDeployment.Labels)
	log.Debugf("Found compliance-operator version %s in namespace %s", version, foundInNamespace)
	if version == "" {
		err = ErrUnableToExtractVersion
	}
	return version, foundInNamespace, err
}

func extractVersionFromLabels(labels map[string]string) string {
	for key, val := range labels {
		// Info: This label is set by OLM, if a custom compliance operator build was deployed via e.g. Helm, this label does not exist.
		if strings.HasSuffix(key, "owner") {
			return strings.TrimPrefix(val, complianceoperator.Name+".")
		}
	}
	return ""
}

func searchForDeployment(ctx context.Context, ns string, dynClient dynamic.Interface) (*appsv1.Deployment, error) {
	// Use cached namespace, if compliance operator deployment was not found search again in all namespaces.
	if ns != "" {
		if complianceOperator, err := getComplianceOperatorDeployment(ns, dynClient, ctx); err == nil {
			return complianceOperator, nil
		}
	}

	// List all namespaces to begin the lookup for compliance operator.
	namespaceList, err := dynClient.Resource(client.NamespaceGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "listing all namespaces")
	}

	for _, namespace := range namespaceList.Items {
		complianceOperator, err := getComplianceOperatorDeployment(namespace.GetName(), dynClient, ctx)
		if err == nil {
			return complianceOperator, nil
		}
		// Until we check all namespaces, we cannot determine if compliance operator is installed or not.
		if kubeAPIErr.IsNotFound(err) {
			continue
		}
		return nil, err
	}

	return nil, errors.Errorf("The %q deployment was not found in any namespace.", complianceoperator.Name)
}

func getComplianceOperatorDeployment(ns string, dynClient dynamic.Interface, ctx context.Context) (*appsv1.Deployment, error) {
	unstructuredObj, err := dynClient.Resource(client.DeploymentGVR).Namespace(ns).Get(ctx, complianceoperator.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "getting compliance operator deployment")
	}
	var deployment appsv1.Deployment
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &deployment); err != nil {
		return nil, errors.Wrap(err, "converting compliance operator deployment")
	}
	return &deployment, nil
}
