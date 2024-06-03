package common

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/util/podutils"
)

var (
	portForwardOnce     sync.Once
	serviceEndpoint     string
	localhostEndpoint   string
	portForwardingError error

	errPortForwarding = errox.ServerError.New("port-forwarding error")
)

const (
	reasonableTimeout = 10 * time.Second
	defaultAPIPort    = 8443
)

// setKubernetesDefaults sets default values on the provided client config for
// accessing the Kubernetes API or returns an error if any of the defaults are
// impossible or invalid.
// The code is taken from:
// k8s.io/kubectl@v0.28.2/pkg/cmd/util/kubectl_match_version.go#L114
func setKubernetesDefaults(config *rest.Config) {
	// This is allowing the GetOptions to be serialized.
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}

	if config.APIPath == "" {
		config.APIPath = "/api"
	}
	if config.NegotiatedSerializer == nil {
		// This codec factory ensures the resources are not converted.
		// Therefore, resources will not be round-tripped through internal
		// versions. Defaulting does not happen on the client.
		config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	}
	if len(config.UserAgent) == 0 {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
}

func sortByPhase(pods []*corev1.Pod) sort.Interface {
	return sort.Reverse(podutils.ActivePods(pods))
}

func getCentralServiceSelectors(ctx context.Context, coreClient coreclient.ServicesGetter, namespace string) (string, error) {
	svc, err := coreClient.Services(namespace).
		Get(ctx, "central", metav1.GetOptions{})
	if err != nil {
		return "", errors.WithMessage(err, "failed to get central service")
	}
	_, sel, err := polymorphichelpers.SelectorsForObject(svc)
	if err != nil {
		return "", errors.WithMessage(err, "failed to get central pod selectors")
	}
	return sel.String(), nil
}

func getCentralPod(ctx context.Context, coreClient coreclient.CoreV1Interface, namespace string) (*corev1.Pod, error) {
	selectors, err := getCentralServiceSelectors(ctx, coreClient, namespace)
	if err != nil {
		return nil, err
	}

	forwardablePod, _, err := polymorphichelpers.GetFirstPod(coreClient,
		namespace, selectors, reasonableTimeout, sortByPhase)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot find a matching pod")
	}

	return coreClient.Pods(namespace). //nolint:wrapcheck
						Get(ctx, forwardablePod.GetName(), metav1.GetOptions{})
}

func getCentralAPIPort(pod *corev1.Pod) int32 {
	for _, container := range pod.Spec.Containers {
		if container.Name == "central" {
			for _, p := range container.Ports {
				if p.Name == "api" {
					return p.ContainerPort
				}
			}
		}
	}
	return defaultAPIPort
}

func getCentralCA(ctx context.Context, coreClient coreclient.CoreV1Interface, namespace string) ([]byte, error) {
	centralTLS, err := coreClient.Secrets(namespace).Get(ctx, "central-tls", metav1.GetOptions{})
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	if ca, ok := centralTLS.Data["ca.pem"]; ok {
		return ca, nil
	}
	return nil, nil
}

func getPortForwarder(restConfig *rest.Config, pod *corev1.Pod, stopChannel <-chan struct{}, readyChannel chan struct{}) (uint16, *portforward.PortForwarder, error) {
	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		return 0, nil, errors.WithMessage(err, "failed to construct k8s REST client")
	}
	// Role rule required:
	//   resources: ["pods/portforward"]
	//   verbs: ["create"]
	req := restClient.Post().Resource(corev1.ResourcePods.String()).
		Namespace(pod.GetNamespace()).Name(pod.GetName()).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return 0, nil, errors.WithMessage(err, "failed to configure k8s REST client transport")
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())
	centralPodPort := getCentralAPIPort(pod)
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("0:%d", centralPodPort)}, stopChannel, readyChannel, nil, nil)
	return uint16(centralPodPort), forwarder, err
}

func getConfigs() (*rest.Config, coreclient.CoreV1Interface, string, error) {
	kubeConfigLoader := genericclioptions.NewConfigFlags(true).ToRawKubeConfigLoader()
	namespace, _, err := kubeConfigLoader.Namespace()
	if err != nil {
		return nil, nil, "", errors.WithMessage(err, "failed to identify central namespace")
	}

	restConfig, err := kubeConfigLoader.ClientConfig()
	if err != nil {
		return nil, nil, "", errors.WithMessage(err, "failed to configure k8s REST client")
	}
	setKubernetesDefaults(restConfig)

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, "", errors.WithMessage(err, "failed to construct k8s client")
	}
	return restConfig, k8sClient.CoreV1(), namespace, nil
}

func runPortForward(ctx context.Context) (string, uint16, error) {
	restConfig, core, namespace, err := getConfigs()
	if err != nil {
		return "", 0, err
	}

	pod, err := getCentralPod(ctx, core, namespace)
	if err != nil {
		return "", 0, errors.WithMessage(err, "cannot get central pod")
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status != corev1.ConditionTrue {
			return "", 0, errors.New("pod is not ready")
		}
	}

	// writing to stopChannel stops the forwarder.
	stopChannel := make(chan struct{}, 1)
	// forwarder writes to readyChannel when ready.
	readyChannel := make(chan struct{})

	// Gracefully stop forwarding on os.Interrupt signal.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	go func() {
		<-signals
		close(stopChannel)
	}()
	centralPort, forwarder, err := getPortForwarder(restConfig, pod, stopChannel, readyChannel)
	if err != nil {
		return "", 0, err
	}
	// Run port forwarder and capture the error.
	errChan := make(chan error)
	go func() {
		errChan <- forwarder.ForwardPorts()
		close(errChan)
	}()

	// Continue if ready, return on error.
	select {
	case <-readyChannel:
	case err := <-errChan:
		if err != nil {
			return "", 0, err
		}
	}
	ports, err := forwarder.GetPorts()
	if err != nil {
		return "", 0, errors.WithMessage(err, "failed to acquire forwarding ports")
	}
	// centralPort is the central pod port, not service port, as forwarding goes
	// directly to the pod, and service name is used to pass TLS validation.
	centralEndpoint := fmt.Sprintf("central.%s:%d", namespace, centralPort)
	return centralEndpoint, ports[0].Local, nil
}

// GetForwardingEndpoint starts port-forwarding to a svc/central pod,
// and returns the service endpoint, to which the requests should be sent,
// and the local endpoint forwarded to the pod, which the dialer should dial to.
func GetForwardingEndpoint() (string, string, error) {
	portForwardOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), reasonableTimeout)
		defer cancel()
		if svc, port, err := runPortForward(ctx); err != nil {
			portForwardingError = errPortForwarding.CausedBy(err)
		} else {
			serviceEndpoint = svc // central.stackrox:8443
			localhostEndpoint = fmt.Sprintf("127.0.0.1:%d", port)
		}
	})
	return serviceEndpoint, localhostEndpoint, portForwardingError
}

// getForwardingDialContext returns a dialer, that resolves central service
// endpoint to the localhost forwarded endpoint: this way the TLS certificate
// validation works.
func getForwardingDialContext() func(ctx context.Context, addr string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		svcEndpoint, localEndpoint, err := GetForwardingEndpoint()
		if err != nil {
			return nil, err
		}
		if addr == svcEndpoint {
			addr = localEndpoint
		}
		return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	}
}
