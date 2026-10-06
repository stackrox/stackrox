package olm

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	api "github.com/stackrox/rox/pkg/operatorbundle/olm/registryapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// catalogSourceGVR is the OLM CatalogSource resource.
var catalogSourceGVR = schema.GroupVersionResource{Group: "operators.coreos.com", Version: "v1alpha1", Resource: "catalogsources"}

// defaultCatalogRegistryPort is the registry gRPC port used when a CatalogSource does not report
// one in status.registryService.
const defaultCatalogRegistryPort = 50051

// ResolveCatalogGRPCAddress determines the in-cluster address of the catalog index gRPC service
// backing the given package. It reads the package's Subscription to find the CatalogSource
// (spec.source / spec.sourceNamespace), then the CatalogSource's status.registryService to build
// "<serviceName>.<namespace>.svc:<port>". This lets Central dial the catalog directly in-cluster,
// without a port-forward.
func ResolveCatalogGRPCAddress(ctx context.Context, dyn dynamic.Interface, pkg string) (string, error) {
	subs, err := dyn.Resource(subGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "listing Subscriptions")
	}
	var source, sourceNS string
	for i := range subs.Items {
		s := &subs.Items[i]
		if nestedString(s, "spec", "name") == pkg {
			source = nestedString(s, "spec", "source")
			sourceNS = nestedString(s, "spec", "sourceNamespace")
			break
		}
	}
	if source == "" || sourceNS == "" {
		return "", errors.Errorf("no Subscription with a CatalogSource found for package %q", pkg)
	}

	cs, err := dyn.Resource(catalogSourceGVR).Namespace(sourceNS).Get(ctx, source, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "getting CatalogSource %s/%s", sourceNS, source)
	}
	svcName := nestedString(cs, "status", "registryService", "serviceName")
	svcNS := nestedString(cs, "status", "registryService", "serviceNamespace")
	if svcNS == "" {
		svcNS = sourceNS
	}
	port, _, _ := unstructured.NestedInt64(cs.Object, "status", "registryService", "port")
	if svcName == "" {
		return "", errors.Errorf("CatalogSource %s/%s has no registryService serviceName in status", sourceNS, source)
	}
	if port == 0 {
		port = defaultCatalogRegistryPort
	}
	return fmt.Sprintf("%s.%s.svc:%d", svcName, svcNS, port), nil
}

// DialCatalogRegistry opens a plaintext gRPC connection to an operator catalog index registry and
// returns a RegistryClient. The OLM CatalogSource registry speaks plaintext gRPC, so no TLS/mTLS
// is used. The caller owns the returned connection and must Close it.
func DialCatalogRegistry(address string) (*grpc.ClientConn, api.RegistryClient, error) {
	conn, err := grpc.NewClient("dns:///"+address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "dialing catalog registry %s", address)
	}
	return conn, api.NewRegistryClient(conn), nil
}
