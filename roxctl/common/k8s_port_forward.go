package common

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/util/podutils"
)

var (
	portForwardOnce     sync.Once
	forwardEndpoint     string
	portForwardingError error

	errPortForwarding = errox.ServerError.New("port-forwarding error")
)

// setKubernetesDefaults sets default values on the provided client config for accessing the
// Kubernetes API or returns an error if any of the defaults are impossible or invalid.
// TODO this isn't what we want.  Each clientset should be setting defaults as it sees fit.
func setKubernetesDefaults(config *restclient.Config) error {
	// TODO remove this hack.  This is allowing the GetOptions to be serialized.
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}

	if config.APIPath == "" {
		config.APIPath = "/api"
	}
	if config.NegotiatedSerializer == nil {
		// This codec factory ensures the resources are not converted. Therefore, resources
		// will not be round-tripped through internal versions. Defaulting does not happen
		// on the client.
		config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	}
	if err := restclient.SetKubernetesDefaults(config); err != nil {
		return errPortForwarding.CausedBy(err)
	}
	return nil
}

func runPortForward() (uint16, error) {
	getter := genericclioptions.NewConfigFlags(true)
	c := getter.ToRawKubeConfigLoader()
	restConfig, err := c.ClientConfig()
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}
	namespace, _, err := c.Namespace()
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}

	if err := setKubernetesDefaults(restConfig); err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}
	coreClient, err := corev1client.NewForConfig(restConfig)
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}
	podClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}

	sortBy := func(pods []*corev1.Pod) sort.Interface { return sort.Reverse(podutils.ActivePods(pods)) }
	forwardablePod, _, err := polymorphichelpers.GetFirstPod(coreClient, namespace, "app=central", 10*time.Second, sortBy)
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}

	pod, err := podClient.CoreV1().Pods(namespace).Get(context.TODO(), forwardablePod.Name, metav1.GetOptions{})
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}
	if pod.Status.Phase != corev1.PodRunning {
		return 0, fmt.Errorf("unable to forward port because pod is not running. Current status=%v", pod.Status.Phase)
	}

	client, err := restclient.RESTClientFor(restConfig)
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)

	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})

	go func() {
		<-signals
		close(stopChannel)
	}()

	req := client.Post().
		Resource("pods").
		Namespace(namespace).
		Name(pod.Name).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}
	streams := genericclioptions.IOStreams{}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	forwarder, err := portforward.New(dialer, []string{"0:8443"}, stopChannel, readyChannel, streams.Out, streams.ErrOut)
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}

	errChan := make(chan error)
	go func() {
		errChan <- forwarder.ForwardPorts()
		close(errChan)
	}()
	select {
	case <-readyChannel:
	case err := <-errChan:
		if err != nil {
			return 0, errPortForwarding.CausedBy(err)
		}
	}

	ports, err := forwarder.GetPorts()
	if err != nil {
		return 0, errPortForwarding.CausedBy(err)
	}
	return ports[0].Local, nil
}

// GetForwardingEndpoint starts port-forwarding to svc/central and returns
// the endpoint with local port forwarding to the service port.
func GetForwardingEndpoint() (string, error) {
	portForwardOnce.Do(func() {
		var port uint16
		port, portForwardingError = runPortForward()
		if portForwardingError == nil {
			forwardEndpoint = fmt.Sprintf("localhost:%d", port)
		}
	})
	return forwardEndpoint, portForwardingError
}
