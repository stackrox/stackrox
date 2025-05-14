package complianceoperator

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/complianceoperator"
	appsv1 "k8s.io/api/apps/v1"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var ErrUnableToExtractVersion = errors.New("compliance operator found " +
	"but labels required for extracting the version are missing")

func GetInstalledVersion(ns string, cli kubernetes.Interface, ctx context.Context) (ver, namespace string, err error) {
	complianceOperatorDeployment, err := searchForDeployment(ns, cli, ctx)
	if err != nil {
		return "", ns, errors.Wrapf(err, "could not find compliance operator deployment %q", ns)
	}

	var version string
	foundInNamespace := complianceOperatorDeployment.GetNamespace()
	for key, val := range complianceOperatorDeployment.Labels {
		// Info: This label is set by OLM, if a custom compliance operator build was deployed via e.g. Helm, this label does not exist.
		if strings.HasSuffix(key, "owner") {
			version = strings.TrimPrefix(val, complianceoperator.Name+".")
			return version, foundInNamespace, nil
		}
	}
	return version, foundInNamespace, ErrUnableToExtractVersion
}

func searchForDeployment(ns string, cli kubernetes.Interface, ctx context.Context) (*appsv1.Deployment, error) {
	// Use cached namespace, if compliance operator deployment was not found search again in all namespaces.
	if ns != "" {
		if complianceOperator, err := getComplianceOperatorDeployment(ns, cli, ctx); err == nil {
			return complianceOperator, nil
		}
	}

	// List all namespaces to begin the lookup for compliance operator.
	namespaceList, err := cli.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaceList.Items {
		complianceOperator, err := getComplianceOperatorDeployment(namespace.GetName(), cli, ctx)
		if err == nil {
			return complianceOperator, nil
		}
		// Until we check all namespaces, we cannot determine if compliance operator is installed or not.
		if kubeAPIErr.IsNotFound(err) {
			continue
		}
		return nil, err
	}

	return nil, errors.Errorf("deployment %s not found in any namespace", complianceoperator.Name)
}

func getComplianceOperatorDeployment(ns string, cli kubernetes.Interface, ctx context.Context) (*appsv1.Deployment, error) {
	return cli.AppsV1().Deployments(ns).Get(ctx, complianceoperator.Name, metav1.GetOptions{})
}
