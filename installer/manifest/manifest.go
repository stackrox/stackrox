package manifest

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

var (
	ReadOnlyMode int32 = 0640
	PostgresUser int64 = 70
	ScannerUser  int64 = 65534
	TwoGigs            = resource.MustParse("2Gi")
	log                = logging.CreateLogger(logging.CurrentModule(), 0)
)

type Resource struct {
	Object        runtime.Object
	Name          string
	IsUpdateable  bool
	ClusterScoped bool
}

type Generator interface {
	Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error)
	Name() string
	Exportable() bool
}

// OrderedGenerator allows specifying the order of generator execution.
// Lower Priority values run first, higher Priority values run last.
// Generators without Priority (standard Generator interface) default to Priority 0.
type OrderedGenerator interface {
	Generator
	Priority() int
}

var central []Generator = []Generator{}
var securedCluster []Generator = []Generator{}
var crs []Generator = []Generator{}

var GeneratorSets map[string]*[]Generator = map[string]*[]Generator{
	"central":        &central,
	"securedcluster": &securedCluster,
	"crs":            &crs,
}

type manifestGenerator struct {
	CA         mtls.CA
	Config     *Config
	Client     *kubernetes.Clientset
	RESTConfig *restclient.Config
}

func New(cfg *Config, clientset *kubernetes.Clientset, restConfig *restclient.Config) (*manifestGenerator, error) {
	if cfg.Namespace == "" {
		return nil, fmt.Errorf("Invalid namespace: %s", cfg.Namespace)
	}

	// Initialize certificate manager before any generators might need it
	InitializeCertificateManager(cfg)

	return &manifestGenerator{
		Config:     cfg,
		Client:     clientset,
		RESTConfig: restConfig,
	}, nil
}

// sortGeneratorsByPriority sorts generators by their priority.
// OrderedGenerators with lower Priority values run first.
// Standard Generators (without Priority) default to Priority 0.
func sortGeneratorsByPriority(generators []Generator) []Generator {
	sorted := make([]Generator, len(generators))
	copy(sorted, generators)

	sort.Slice(sorted, func(i, j int) bool {
		iPriority := 0
		if orderedGen, ok := sorted[i].(OrderedGenerator); ok {
			iPriority = orderedGen.Priority()
		}

		jPriority := 0
		if orderedGen, ok := sorted[j].(OrderedGenerator); ok {
			jPriority = orderedGen.Priority()
		}

		return iPriority < jPriority
	})

	return sorted
}

func (m *manifestGenerator) Export(ctx context.Context, generators []Generator) error {
	for _, generator := range sortGeneratorsByPriority(generators) {
		resources, err := generator.Generate(ctx, m)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failure generating %s", generator.Name()))
		}
		for _, resource := range resources {
			gvk := resource.Object.GetObjectKind().GroupVersionKind()
			objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource.Object)
			if err != nil {
				panic(err)
			}

			md := objMap["metadata"].(map[string]interface{})

			// Add a label to make it easier to clean up all the resources
			labels, exists := objMap["labels"]
			if !exists {
				labels = map[string]string{}
				objMap["labels"] = labels
			}
			labels.(map[string]string)["system"] = "stackrox"

			delete(md, "creationTimestamp")
			delete(objMap, "status")

			var buf bytes.Buffer
			encoder := yaml.NewEncoder(&buf)
			defer encoder.Close()
			encoder.SetIndent(2)

			if err := encoder.Encode(objMap); err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error encoding resource %s/%s from generator %s into yaml", gvk.Kind, resource.Name, generator.Name()))
			}

			println("-----")
			print(string(buf.Bytes()))
		}
	}
	return nil
}

func (m *manifestGenerator) Apply(ctx context.Context, generators []Generator) error {
	dynamicClient, err := dynamic.NewForConfig(m.RESTConfig)
	if err != nil {
		panic(err)
	}

	for _, generator := range sortGeneratorsByPriority(generators) {
		log.Info("-----")
		log.Infof("Generating resources from %s...", generator.Name())
		resources, err := generator.Generate(ctx, m)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failure generating %s", generator.Name()))
		}

		if len(resources) == 0 {
			log.Infof("No resources to apply from %s", generator.Name())
			continue
		}

		log.Infof("Applying resources from %s...", generator.Name())
		for _, resource := range resources {
			gvk := resource.Object.GetObjectKind().GroupVersionKind()

			var dynClient dynamic.ResourceInterface
			if resource.ClusterScoped {
				dynClient = dynamicClient.Resource(toGVR(gvk))
			} else {
				dynClient = dynamicClient.Resource(toGVR(gvk)).Namespace(m.Config.Namespace)
			}

			objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource.Object)
			if err != nil {
				panic(err)
			}

			unst := unstructured.Unstructured{Object: objMap}
			_, err = dynClient.Create(ctx, &unst, metav1.CreateOptions{})

			if err != nil {
				if k8serrors.IsAlreadyExists(err) {
					if resource.IsUpdateable {
						_, err := dynClient.Update(ctx, &unst, metav1.UpdateOptions{})
						if err != nil {
							return errors.Wrap(err, fmt.Sprintf("Failed to update %s/%s", gvk.Kind, resource.Name))
						} else {
							log.Infof("Updated %s/%s", gvk.Kind, resource.Name)
						}
					} else {
						log.Infof("Skipped %s/%s", gvk.Kind, resource.Name)
					}
				} else {
					return errors.Wrap(err, fmt.Sprintf("Failed to create %s/%s", gvk.Kind, resource.Name))
				}
			} else {
				log.Infof("Created %s/%s", gvk.Kind, resource.Name)
			}
		}
	}
	return nil
}

type tlsCallback func(fileMap map[string][]byte) error

func genTlsSecret(name string, ca mtls.CA, issueCert tlsCallback) (Resource, error) {
	fileMap := make(map[string][]byte)
	certgen.AddCACertToFileMap(fileMap, ca)

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: fileMap,
	}
	secret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))

	if err := issueCert(fileMap); err != nil {
		return Resource{}, fmt.Errorf("issuing certificate for %s: %w", name, err)
	}

	return Resource{
		Object:       secret,
		Name:         name,
		IsUpdateable: false,
	}, nil
}

func genClusterRoleBinding(serviceAccountName, roleName, ns string) Resource {
	name := fmt.Sprintf("%s-%s-%s", ns, serviceAccountName, roleName)
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccountName,
			Namespace: ns,
		}},
	}
	binding.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"))

	return Resource{
		Object:        binding,
		Name:          name,
		IsUpdateable:  true,
		ClusterScoped: true,
	}
}

func genRole(name string, rules []rbacv1.PolicyRule) Resource {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: rules,
	}
	role.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("Role"))
	return Resource{
		Object:       role,
		Name:         name,
		IsUpdateable: true,
	}
}

func genRoleBinding(serviceAccountName, roleName, ns string) Resource {
	name := fmt.Sprintf("%s-%s", serviceAccountName, roleName)
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccountName,
			Namespace: ns,
		}},
	}
	binding.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("RoleBinding"))
	return Resource{
		Object:       binding,
		Name:         name,
		IsUpdateable: true,
	}
}

func genService(name string, ports []v1.ServicePort) Resource {
	svc := v1.Service{
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: ports,
		},
	}
	svc.SetName(name)
	svc.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))
	return Resource{
		Object:       &svc,
		Name:         name,
		IsUpdateable: true,
	}
}

func genServiceAccount(name string) Resource {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	sa.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ServiceAccount"))
	return Resource{
		Object:       sa,
		Name:         name,
		IsUpdateable: false,
	}
}

func toGVR(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind + "s"),
	}
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
	if spec != nil {
		v.Volume.Name = v.Name
		spec.Volumes = append(spec.Volumes, v.Volume)
	}
}

func RestrictedSecurityContext(user int64) *v1.SecurityContext {
	FALSE := false
	TRUE := true
	return &v1.SecurityContext{
		RunAsUser:                &user,
		RunAsGroup:               &user,
		AllowPrivilegeEscalation: &FALSE,
		SeccompProfile: &v1.SeccompProfile{
			Type: v1.SeccompProfileTypeRuntimeDefault,
		},
		RunAsNonRoot: &TRUE,
		Capabilities: &v1.Capabilities{
			Drop: []v1.Capability{"ALL"},
		},
	}
}
