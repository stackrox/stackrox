package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func NoopReconcileExtension(context.Context, *unstructured.Unstructured, func(statusFunc extensions.UpdateStatusFunc), logr.Logger) error {
	return nil
}
