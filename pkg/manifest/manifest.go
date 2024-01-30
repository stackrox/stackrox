package manifest

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	ReadOnlyMode int32 = 0640
	PostgresUser int64 = 70
	CentralUser  int64 = 4000
	ScannerUser  int64 = 65534
	TwoGigs            = resource.MustParse("2Gi")
	log                = logging.CreateLogger(logging.CurrentModule(), 0)
)

type ManifestGenerator struct {
	CA        mtls.CA
	Namespace string
}

func New(ns string) (*ManifestGenerator, error) {
	if ns == "" {
		return nil, fmt.Errorf("Invalid namespace: %s", ns)
	}

	ca, err := certgen.GenerateCA()

	if err != nil {
		return nil, fmt.Errorf("creating new CA: %w\n", err)
	}

	return &ManifestGenerator{
		Namespace: ns,
		CA:        ca,
	}, nil
}

func (m ManifestGenerator) Apply(ctx context.Context, clientset *kubernetes.Clientset) error {
	if err := m.applyNamespace(ctx, clientset); err != nil {
		panic(err)
	}

	if err := m.applyCentral(ctx, clientset); err != nil {
		panic(err)
	}

	if err := m.applyScanner(ctx, clientset); err != nil {
		panic(err)
	}

	return nil
}

func (m ManifestGenerator) applyNamespace(ctx context.Context, clientset *kubernetes.Clientset) error {
	ns := v1.Namespace{}
	ns.SetName(m.Namespace)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})

	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create namespace: %w\n", err)
	}

	log.Info("Created namespace")

	return nil
}

type VolumeDefAndMount struct {
	Name      string
	MountPath string
	ReadOnly  bool
	Volume    v1.Volume
}

func (v VolumeDefAndMount) Apply(c *v1.Container, spec *v1.PodSpec) {
	c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
		Name:      v.Name,
		MountPath: v.MountPath,
		ReadOnly:  v.ReadOnly,
	})
	v.Volume.Name = v.Name
	spec.Volumes = append(spec.Volumes, v.Volume)
}
