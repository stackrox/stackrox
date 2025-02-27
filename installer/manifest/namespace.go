package manifest

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

type NamespaceGenerator struct{}

func (g NamespaceGenerator) Name() string {
	return "Namespace"
}

func (g NamespaceGenerator) Exportable() bool {
	return true
}

func (g NamespaceGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	ns := v1.Namespace{}
	ns.SetName(m.Config.Namespace)

	ns.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Namespace"))

	return []Resource{{
		Object:        &ns,
		Name:          ns.Name,
		IsUpdateable:  false,
		ClusterScoped: true,
	}}, nil
}

func init() {
	genList := []Generator{NamespaceGenerator{}}
	central = append(genList, central...)
	securedCluster = append(genList, securedCluster...)
}
