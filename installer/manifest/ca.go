package manifest

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/certgen"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CAGenerator struct{}

func (g CAGenerator) Name() string {
	return "Certificate Authority (CA)"
}

func (g CAGenerator) Exportable() bool {
	return true
}

func (g CAGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	if m.Config.Action == "apply" {
		g.GetCA(ctx, m)
		if m.CA != nil {
			log.Info("CA already exists")
			return []Resource{}, nil
		}
	}

	ca, err := certgen.GenerateCA()
	if err != nil {
		return []Resource{}, fmt.Errorf("Error generating CA: %v", err)
	}

	fileMap := make(map[string][]byte)
	certgen.AddCAToFileMap(fileMap, ca)

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "additional-ca",
		},
		Data: fileMap,
	}
	secret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))

	m.CA = ca

	return []Resource{{
		Object:       secret,
		Name:         secret.Name,
		IsUpdateable: false,
	}}, nil
}

func (g *CAGenerator) GetCA(ctx context.Context, m *manifestGenerator) error {
	secret, err := m.Client.CoreV1().Secrets(m.Config.Namespace).Get(ctx, "additional-ca", metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("Error fetching additional-ca secret: %w", err)
	}

	ca, err := certgen.LoadCAFromFileMap(secret.Data)
	if err != nil {
		return fmt.Errorf("Error loading CA from additional-ca secret: %v", err)
	}

	m.CA = ca
	return nil
}

func init() {
	central = append(central, CAGenerator{})
	crs = append(crs, CAGenerator{})
}
