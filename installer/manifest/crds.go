package manifest

import (
	"context"
	_ "embed"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

//go:embed crds/config.stackrox.io_securitypolicies.yaml
var securityPolicyCRD []byte

type CRDGenerator struct {
	crdYAML []byte
	name    string
}

func NewCRDGenerator(name string, crdYAML []byte) CRDGenerator {
	return CRDGenerator{
		crdYAML: crdYAML,
		name:    name,
	}
}

func (g CRDGenerator) Name() string {
	return fmt.Sprintf("%s CRD", g.name)
}

func (g CRDGenerator) Exportable() bool {
	return true
}

func (g CRDGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	scheme := runtime.NewScheme()
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add CRD types to scheme: %w", err)
	}

	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	obj, _, err := decoder.Decode(g.crdYAML, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CRD YAML: %w", err)
	}

	crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		return nil, fmt.Errorf("decoded object is not a CRD, got %T", obj)
	}

	return []Resource{{
		Object:        crd,
		Name:          crd.ObjectMeta.Name,
		IsUpdateable:  false,
		ClusterScoped: true,
	}}, nil
}

func init() {
	central = append(central, NewCRDGenerator("SecurityPolicy", securityPolicyCRD))
}
