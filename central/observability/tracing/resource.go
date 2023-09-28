package tracing

import (
	"os"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func CentralResource() *resource.Resource {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("central"),
			semconv.ServiceVersion(version.GetMainVersion()),
			semconv.K8SPodName(os.Getenv("HOSTNAME")),
			semconv.K8SNamespaceName(os.Getenv("POD_NAMESPACE")),
		),
	)
	utils.Should(err)
	return r
}
