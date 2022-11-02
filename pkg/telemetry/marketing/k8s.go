package marketing

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func getK8SData() (*Device, error) {
	rc, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "cannot create k8s config")
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create k8s clientset")
	}
	v, err := clientset.ServerVersion()
	if err != nil {
		return nil, err
	}
	di := clientset.AppsV1().Deployments("stackrox")
	opts := v1.GetOptions{}
	d, err := di.Get(context.Background(), "central", opts)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get central deployment")
	}
	paths := d.GetAnnotations()["stackrox.com/telemetry-apipaths"]

	return &Device{
		ID:       string(d.GetUID()),
		Version:  v.GitVersion,
		ApiPaths: strings.Split(paths, ","),
	}, nil
}
