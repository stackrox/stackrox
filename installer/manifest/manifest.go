package manifest

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	stackroxv1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/utils"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

var (
	ReadOnlyMode int32 = 0640
	PostgresUser int64 = 70
	ScannerUser  int64 = 65534
	TwoGigs            = resource.MustParse("2Gi")
	log                = logging.CreateLogger(logging.CurrentModule(), 0)
)

type Resource struct {
	Object       runtime.Object
	Name         string
	IsUpdateable bool
}

type Generator interface {
	Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error)
	Name() string
	Exportable() bool
}

var central []Generator = []Generator{}
var securedCluster []Generator = []Generator{}

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

	return &manifestGenerator{
		Config:     cfg,
		Client:     clientset,
		RESTConfig: restConfig,
	}, nil
}

func (m *manifestGenerator) Generate(ctx context.Context) error {
	if err := m.applyNamespace(ctx); err != nil {
		panic(err)
	}

	dynamicClient, err := dynamic.NewForConfig(m.RESTConfig)
	if err != nil {
		panic(err)
	}

	if err := m.getCA(ctx); err != nil {
		return fmt.Errorf("Getting CA: %w\n", err)
	}

	if m.CA == nil {
		central = append([]Generator{CertGenerator{}}, central...)
	}

	for _, generator := range central {
		resources, err := generator.Generate(ctx, m)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failure generating %s", generator.Name()))
		}
		for _, resource := range resources {
			objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource.Object)
			if err != nil {
				panic(err)
			}

			unst := unstructured.Unstructured{Object: objMap}

			gvk := resource.Object.GetObjectKind().GroupVersionKind()
			dynClient := dynamicClient.Resource(toGVR(gvk)).Namespace(m.Config.Namespace)
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

func (m *manifestGenerator) getCA(ctx context.Context) error {
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

// func (m *manifestGenerator) Apply(ctx context.Context) error {

// 	if err := m.getCA(ctx); err != nil {
// 		return fmt.Errorf("Getting CA: %w\n", err)
// 	}

// 	if err := m.createNonrootV2SCCRole(ctx); err != nil {
// 		panic(err)
// 	}

// 	if err := m.applyCentral(ctx); err != nil {
// 		panic(err)
// 	}

// 	if m.Config.ScannerV4 {
// 		if err := m.applyScannerV4(ctx); err != nil {
// 			panic(err)
// 		}
// 	}

// 	if err := m.applyScanner(ctx); err != nil {
// 		panic(err)
// 	}

// 	if err := m.applyCRS(ctx); err != nil {
// 		log.Errorf("Failed to apply CRS: %v", err)
// 		return nil
// 	}

// 	if err := m.applySensor(ctx); err != nil {
// 		panic(err)
// 	}

// 	return nil
// }

type CertGenerator struct{}

func (g CertGenerator) Name() string {
	return "Certificate"
}

func (g CertGenerator) Exportable() bool {
	return true
}

func (g CertGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
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

func (m *manifestGenerator) applyNamespace(ctx context.Context) error {
	ns := v1.Namespace{}
	ns.SetName(m.Config.Namespace)
	_, err := m.Client.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})

	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create namespace: %w\n", err)
	}

	log.Info("Created namespace")

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

func (m *manifestGenerator) applyTlsSecret(ctx context.Context, name string, issueCert tlsCallback) error {
	secret, err := m.Client.CoreV1().Secrets(m.Config.Namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("Error fetching %s secret: %w", name, err)
	}

	if err == nil {
		log.Infof("Secret %s already found.", name)
		return nil
	}

	fileMap := make(map[string][]byte)
	certgen.AddCACertToFileMap(fileMap, m.CA)

	secret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: fileMap,
	}

	if err = issueCert(fileMap); err != nil {
		return fmt.Errorf("issuing certificate for %s: %w", name, err)
	}

	_, err = m.Client.CoreV1().Secrets(m.Config.Namespace).Create(ctx, secret, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
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
	_, err := m.Client.RbacV1().Roles(m.Config.Namespace).Create(ctx, &role, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Infof("Role %s already exists", name)
		} else {
			return fmt.Errorf("Error creating role %s: %w", name, err)
		}
	}

	return nil
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

func (m *manifestGenerator) createRoleBinding(ctx context.Context, serviceAccountName, roleName string) error {
	name := fmt.Sprintf("%s-%s", serviceAccountName, roleName)
	roleBinding := rbacv1.RoleBinding{
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
			Namespace: m.Config.Namespace,
		}},
	}
	_, err := m.Client.RbacV1().RoleBindings(m.Config.Namespace).Create(ctx, &roleBinding, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Infof("Role binding %s already exists", name)
			if _, err := m.Client.RbacV1().RoleBindings(m.Config.Namespace).Update(ctx, &roleBinding, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("Error updating role binding %s: %w", name, err)
			}
		} else {
			return fmt.Errorf("Error creating role binding %s: %w", name, err)
		}
	}

	return nil
}

func (m *manifestGenerator) createClusterRoleBinding(ctx context.Context, serviceAccountName, roleName string) error {
	name := fmt.Sprintf("%s-%s-%s", m.Config.Namespace, serviceAccountName, roleName)
	roleBinding := rbacv1.ClusterRoleBinding{
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
			Namespace: m.Config.Namespace,
		}},
	}
	_, err := m.Client.RbacV1().ClusterRoleBindings().Create(ctx, &roleBinding, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Infof("Role binding %s already exists, skipping", name)
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

	_, err := m.Client.CoreV1().ServiceAccounts(m.Config.Namespace).Create(ctx, &acct, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Infof("Service account %s already exists", name)
		} else {
			return fmt.Errorf("Error creating service account %s: %w", name, err)
		}
	}

	return nil
}

func (m *manifestGenerator) applyConfigMap(ctx context.Context, name string, cm *v1.ConfigMap) error {
	cm.SetName(name)
	_, err := m.Client.CoreV1().ConfigMaps(m.Config.Namespace).Create(ctx, cm, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = m.Client.CoreV1().ConfigMaps(m.Config.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("Error updating configmap %s: %w", name, err)
			}
		} else {
			return fmt.Errorf("Error creating configmap %s: %w", name, err)
		}
	}

	return nil
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

func (m *manifestGenerator) applyService(ctx context.Context, name string, ports []v1.ServicePort) error {
	svc := &v1.Service{
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: ports,
		},
	}
	svc.SetName(name)
	_, err := m.Client.CoreV1().Services(m.Config.Namespace).Create(ctx, svc, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = m.Client.CoreV1().Services(m.Config.Namespace).Update(ctx, svc, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("Error updating service %s: %w", name, err)
			}
		} else {
			return fmt.Errorf("Error creating service %s: %w", name, err)
		}
	}

	return nil
}

func getRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0") // 0 means a random port will be assigned
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	return strconv.Atoi(port)
}

func (m *manifestGenerator) portForward(ctx context.Context, port int) (chan struct{}, error) {
	pods, err := m.Client.CoreV1().Pods(m.Config.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": "central"}}),
	})
	if err != nil {
		return nil, err
	} else if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no central pods found: %v", err)
	}

	centralPod := pods.Items[0]

	if centralPod.Status.Phase != v1.PodRunning {
		return nil, errors.New("Pod not yet ready")
	}

	podName := centralPod.Name

	if podName == "" {
		return nil, errors.New("Timed out waiting for pod to start running")
	}

	req := m.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(m.Config.Namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(m.RESTConfig)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())
	if err != nil {
		return nil, fmt.Errorf("Failed to create websocket: %v", err)
	}

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	pf, err := portforward.New(dialer, []string{fmt.Sprintf("%d:8443", port)}, stopChan, readyChan, os.Stdout, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create tunnel for port-forwarding: %v", err)
	}

	go func() {
		err = pf.ForwardPorts()
	}()

	log.Info("Waiting for port forwarder to become ready...")

	select {
	case <-readyChan:
		log.Info("port forwarder ready!")
		return stopChan, nil
	case <-time.After(5 * time.Second):
		errMsg := "Timed out waiting for port forwarding to start"
		if err != nil {
			return nil, errors.Wrap(err, errMsg)
		}
		return nil, errors.New(errMsg)
	}
}

func (m *manifestGenerator) centralConnection(ctx context.Context, port int) (*grpc.ClientConn, error) {
	clientconn.SetUserAgent("Installer")

	dialOpts := []grpc.DialOption{
		grpc.WithNoProxy(),
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(m.CA.CertPEM())

	opts := clientconn.Options{
		InsecureNoTLS:                  false,
		InsecureAllowCredsViaPlaintext: false,
		DialOptions:                    dialOpts,
		TLS: clientconn.TLSConfigOptions{
			RootCAs: pool,
		},
	}

	opts.ConfigureBasicAuth("admin", "letmein")

	callOpts := []grpc.CallOption{grpc.MaxCallRecvMsgSize(12 * size.MB)}
	centralHostPort := fmt.Sprintf("localhost:%d", port)
	return clientconn.GRPCConnection(ctx, mtls.CentralSubject, centralHostPort, opts, grpc.WithDefaultCallOptions(callOpts...))
}

func (m *manifestGenerator) applyCRS(ctx context.Context) error {
	sec, err := m.Client.CoreV1().Secrets(m.Config.Namespace).Get(ctx, "cluster-registration-secret", metav1.GetOptions{})
	if sec != nil && err == nil {
		log.Info("CRS already exists")
		return nil
	} else if !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, "Failed to fetch cluster-registration-secret")
	}

	port, err := getRandomPort()
	if err != nil {
		panic(err)
	}

	var resp *stackroxv1.CRSGenResponse
	retry.WithRetry(func() error {
		pfCloseChan, err := m.portForward(ctx, port)
		if err != nil {
			return errors.Wrap(err, "Failed to create port forwarder")
		}
		defer close(pfCloseChan)

		log.Info("Creating grpc connection to central...")

		conn, err := m.centralConnection(ctx, port)
		if err != nil {
			return errors.Wrap(err, "Failed to create gRPC connection")
		}
		defer utils.IgnoreError(conn.Close)

		log.Info("Invoking CRS endpoint...")
		svc := stackroxv1.NewClusterInitServiceClient(conn)
		req := stackroxv1.CRSGenRequest{Name: "local"}
		resp, err = svc.GenerateCRS(ctx, &req)
		if err != nil {
			if errStatus, ok := status.FromError(err); ok && errStatus.Code() == codes.Unimplemented {
				return errors.Wrap(err, "missing CRS support in Central")
			}
			return errors.Wrap(err, "generating new CRS")
		}
		return nil
	}, retry.Tries(30), retry.BetweenAttempts(func(previousAttemptNumber int) {
		log.Info("Waiting for central endpoint to start listening")
		time.Sleep(5 * time.Second)
	}))

	crs := extractCRS(resp.GetCrs())

	if crs == "" {
		return errors.New("Failed to extract CRS")
	}

	data, err := base64.StdEncoding.DecodeString(crs)
	if err != nil {
		return errors.Wrap(err, "Failed to base64 decode CRS")
	}

	crsSecret := v1.Secret{
		Data: map[string][]byte{"crs": []byte(data)},
	}
	crsSecret.SetName("cluster-registration-secret")
	if _, err := m.Client.CoreV1().Secrets(m.Config.Namespace).Create(ctx, &crsSecret, metav1.CreateOptions{}); err != nil {
		return errors.Wrap(err, "Failed to create CRS secret")
	}
	log.Info("Created cluster-registration-secret")

	return nil
}

func extractCRS(input []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(input)))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "  crs: ") {
			return strings.TrimPrefix(line, "  crs: ")
		}
	}
	return ""
}

func toGVR(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind + "s"),
	}
}
