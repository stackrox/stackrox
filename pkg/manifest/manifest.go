package manifest

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

type manifestGenerator struct {
	CA        mtls.CA
	Namespace string
	Client    *kubernetes.Clientset
}

func New(ns string, clientset *kubernetes.Clientset) (*manifestGenerator, error) {
	if ns == "" {
		return nil, fmt.Errorf("Invalid namespace: %s", ns)
	}

	return &manifestGenerator{
		Namespace: ns,
		Client:    clientset,
	}, nil
}

func (m *manifestGenerator) Apply(ctx context.Context) error {
	if err := m.applyNamespace(ctx); err != nil {
		panic(err)
	}

	if err := m.getCA(ctx); err != nil {
		return fmt.Errorf("Getting CA: %w\n", err)
	}

	if err := m.createNonrootV2SCCRole(ctx); err != nil {
		panic(err)
	}

	if err := m.applyCentral(ctx); err != nil {
		panic(err)
	}

	if err := m.applyScanner(ctx); err != nil {
		panic(err)
	}

	return nil
}

func (m *manifestGenerator) getCA(ctx context.Context) error {
	var ca mtls.CA
	var err error
	secret, err := m.Client.CoreV1().Secrets(m.Namespace).Get(ctx, "additional-ca", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			ca, err = certgen.GenerateCA()
			if err != nil {
				return fmt.Errorf("Error generating CA: %v", err)
			}

			fileMap := make(types.SecretDataMap)
			certgen.AddCAToFileMap(fileMap, ca)

			secret = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "additional-ca",
				},
				Data: fileMap,
			}

			_, err = m.Client.CoreV1().Secrets(m.Namespace).Create(ctx, secret, metav1.CreateOptions{})
			if err != nil {
				if errors.IsAlreadyExists(err) {
					return fmt.Errorf("Race condition: Secret additional-ca got created just before now: %w", err)
				}
				return fmt.Errorf("Error creating secret additional-ca: %w", err)
			}
		} else {
			return fmt.Errorf("Error fetching additional-ca secret: %w", err)
		}
	} else {
		ca, err = certgen.LoadCAFromFileMap(secret.Data)
		if err != nil {
			return fmt.Errorf("Error loading CA from additional-ca secret: %v", err)
		}
	}

	m.CA = ca

	return nil
}

func (m *manifestGenerator) applyNamespace(ctx context.Context) error {
	ns := v1.Namespace{}
	ns.SetName(m.Namespace)
	_, err := m.Client.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})

	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create namespace: %w\n", err)
	}

	log.Info("Created namespace")

	return nil
}

type tlsCallback func(fileMap types.SecretDataMap) error

func (m *manifestGenerator) applyTlsSecret(ctx context.Context, name string, issueCert tlsCallback) error {
	secret, err := m.Client.CoreV1().Secrets(m.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		log.Infof("Secret %s already found.", name)
		return nil
	}

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Error fetching %s secret: %w", name, err)
	}

	fileMap := make(types.SecretDataMap)
	certgen.AddCAToFileMap(fileMap, m.CA)

	secret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: fileMap,
	}

	if err = issueCert(fileMap); err != nil {
		return fmt.Errorf("issuing certificate for %s: %w", name, err)
	}

	_, err = m.Client.CoreV1().Secrets(m.Namespace).Create(ctx, secret, metav1.CreateOptions{})

	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("Race condition: Secret %s got created just before now: %w", name, err)
		}
		return fmt.Errorf("Error creating secret %s: %w", name, err)
	}

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

func (m *manifestGenerator) createNonrootV2SCCRole(ctx context.Context) error {
	name := "use-nonroot-v2-scc"
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups:     []string{"security.openshift.io"},
			Resources:     []string{"securitycontextconstraints"},
			ResourceNames: []string{"nonroot-v2"},
			Verbs:         []string{"use"},
		}},
	}
	_, err := m.Client.RbacV1().Roles(m.Namespace).Create(ctx, &role, metav1.CreateOptions{})

	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Infof("Role %s already exists", name)
		} else {
			return fmt.Errorf("Error creating role %s: %w", name, err)
		}
	}

	return nil
}

func (m *manifestGenerator) createNonrootV2SCCRoleBinding(ctx context.Context, serviceAccountName string) error {
	name := fmt.Sprintf("%s-use-nonroot-v2-scc", serviceAccountName)
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "use-nonroot-v2-scc",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccountName,
			Namespace: "stackrox",
		}},
	}
	_, err := m.Client.RbacV1().RoleBindings(m.Namespace).Create(ctx, &roleBinding, metav1.CreateOptions{})

	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Infof("Role binding %s already exists", name)
		} else {
			return fmt.Errorf("Error creating role binding %s: %w", name, err)
		}
	}

	return nil
}

func (m *manifestGenerator) createServiceAccount(ctx context.Context, name string) error {
	acct := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := m.Client.CoreV1().ServiceAccounts(m.Namespace).Create(ctx, &acct, metav1.CreateOptions{})

	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Infof("Service account %s already exists", name)
		} else {
			return fmt.Errorf("Error creating service account %s: %w", name, err)
		}
	}

	return m.createNonrootV2SCCRoleBinding(ctx, name)
}
