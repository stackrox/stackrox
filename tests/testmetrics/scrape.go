package testmetrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// Transport selects how pod metrics are scraped.
type Transport string

const (
	TransportProxy       Transport = "proxy"
	TransportPortForward Transport = "portforward"
)

// podSnapshot is a single /metrics (or custom path) scrape from one pod.
type podSnapshot struct {
	podName string
	body    string
	err     error
}

// podCollectOptions scrapes metrics from every pod matching LabelSelector.
type podCollectOptions struct {
	namespace     string
	labelSelector string
	fieldSelector string
	port          int
	metricsPath   string
	transport     Transport
	restConfig    *rest.Config
}

// ScrapeTarget describes how to collect metrics for one Kubernetes component.
type ScrapeTarget struct {
	ComponentName string
	Namespace     string
	LabelSelector string
	FieldSelector string
	MetricsPort   int
	MetricsPath   string
}

// cleanMetricsPath returns a slash-trimmed path, defaulting to "metrics".
func cleanMetricsPath(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "metrics"
	}
	return path
}

// scrapePodViaProxy scrapes Prometheus text from a pod via GET .../pods/{name}[:port]/proxy/{path}.
func scrapePodViaProxy(ctx context.Context, clientset kubernetes.Interface, namespace, podName string, port int, metricsPath string) ([]byte, error) {
	if namespace == "" || podName == "" {
		return nil, errors.New("testmetrics: scrapePodViaProxy: namespace and pod name are required")
	}
	path := cleanMetricsPath(metricsPath)
	segments := strings.Split(path, "/")
	podSubresourceName := podName
	if port > 0 {
		podSubresourceName = fmt.Sprintf("%s:%d", podName, port)
	}
	req := clientset.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Resource("pods").
		Name(podSubresourceName).
		SubResource("proxy")
	for _, seg := range segments {
		if seg != "" {
			req = req.Suffix(seg)
		}
	}
	return req.Do(ctx).Raw()
}

// scrapePodViaPortForward opens an ephemeral port-forward to remotePort on the pod
// and GETs metricsPath on localhost.
func scrapePodViaPortForward(ctx context.Context, clientset kubernetes.Interface, restCfg *rest.Config, namespace, podName string, remotePort int, metricsPath string) ([]byte, error) {
	if restCfg == nil {
		return nil, errors.New("testmetrics: scrapePodViaPortForward: rest.Config is required")
	}
	if namespace == "" || podName == "" {
		return nil, errors.New("testmetrics: scrapePodViaPortForward: namespace and pod name are required")
	}
	if remotePort <= 0 {
		return nil, errors.New("testmetrics: scrapePodViaPortForward: remote metrics port must be positive")
	}
	path := "/" + cleanMetricsPath(metricsPath)

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").Namespace(namespace).Name(podName).SubResource("portforward")
	transport, upgrader, err := spdy.RoundTripperFor(restCfg)
	if err != nil {
		return nil, fmt.Errorf("testmetrics: port-forward transport: %w", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())

	stopCh := make(chan struct{})
	readyCh := make(chan struct{})
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("0:%d", remotePort)}, stopCh, readyCh, io.Discard, io.Discard)
	if err != nil {
		return nil, fmt.Errorf("testmetrics: port-forward: %w", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- forwarder.ForwardPorts() }()

	select {
	case <-ctx.Done():
		close(stopCh)
		<-errCh
		return nil, ctx.Err()
	case err = <-errCh:
		return nil, fmt.Errorf("testmetrics: port-forward to %s/%s:%d failed: %w", namespace, podName, remotePort, err)
	case <-readyCh:
	}

	ports, err := forwarder.GetPorts()
	if err != nil || len(ports) == 0 {
		close(stopCh)
		<-errCh
		return nil, fmt.Errorf("testmetrics: port-forward: no local ports: %w", err)
	}
	localPort := int(ports[0].Local)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", localPort, path), nil)
	if err != nil {
		close(stopCh)
		<-errCh
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		close(stopCh)
		<-errCh
		return nil, fmt.Errorf("testmetrics: port-forward metrics GET: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(resp.Body)
	close(stopCh)
	<-errCh
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("testmetrics: port-forward metrics HTTP %s", resp.Status)
	}
	return body, nil
}

func scrapePod(ctx context.Context, clientset kubernetes.Interface, opts podCollectOptions, podName string) ([]byte, error) {
	transport := opts.transport
	if transport == "" {
		transport = TransportProxy
	}
	switch transport {
	case TransportPortForward:
		if opts.restConfig == nil {
			return nil, errors.New("testmetrics: podCollectOptions.restConfig is required for port-forward transport")
		}
		if opts.port <= 0 {
			return nil, errors.New("testmetrics: podCollectOptions.port must be set for port-forward transport")
		}
		return scrapePodViaPortForward(ctx, clientset, opts.restConfig, opts.namespace, podName, opts.port, opts.metricsPath)
	case TransportProxy:
		return scrapePodViaProxy(ctx, clientset, opts.namespace, podName, opts.port, opts.metricsPath)
	default:
		return nil, fmt.Errorf("testmetrics: unsupported metrics transport %q", transport)
	}
}

// collectFromPods lists pods matching labelSelector/fieldSelector and scrapes metrics
// from each (same port/path/transport).
func collectFromPods(ctx context.Context, clientset kubernetes.Interface, opts podCollectOptions) ([]podSnapshot, error) {
	if opts.namespace == "" {
		return nil, errors.New("testmetrics: collectFromPods: namespace is required")
	}
	if opts.transport == TransportPortForward && opts.restConfig == nil {
		return nil, errors.New("testmetrics: collectFromPods: restConfig required for port-forward")
	}
	list, err := clientset.CoreV1().Pods(opts.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: opts.labelSelector,
		FieldSelector: opts.fieldSelector,
	})
	if err != nil {
		return nil, err
	}
	out := make([]podSnapshot, 0, len(list.Items))
	for i := range list.Items {
		name := list.Items[i].Name
		body, err := scrapePod(ctx, clientset, opts, name)
		snap := podSnapshot{podName: name}
		if err != nil {
			snap.err = err
		} else {
			snap.body = string(body)
		}
		out = append(out, snap)
	}
	return out, nil
}

// ScrapeComponent scrapes pods of a single component and parses the requested counters.
// Pods that fail to serve metrics are skipped; an error is returned only when no pod yields valid data.
func ScrapeComponent(ctx context.Context, clientset kubernetes.Interface, target ScrapeTarget, transport Transport, restCfg *rest.Config, queries []Query) (map[string]Value, error) {
	snaps, err := collectFromPods(ctx, clientset, podCollectOptions{
		namespace:     target.Namespace,
		labelSelector: target.LabelSelector,
		fieldSelector: target.FieldSelector,
		port:          target.MetricsPort,
		metricsPath:   target.MetricsPath,
		transport:     transport,
		restConfig:    restCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("scrape %s: %w", target.ComponentName, err)
	}
	if len(snaps) == 0 {
		return nil, fmt.Errorf("scrape %s: no pods found (selector=%q field=%q)",
			target.ComponentName, target.LabelSelector, target.FieldSelector)
	}
	var b strings.Builder
	okPods := 0
	var podErrors []string
	for _, s := range snaps {
		if s.err != nil {
			podErrors = append(podErrors, fmt.Sprintf("pod %s: %v", s.podName, s.err))
			continue
		}
		okPods++
		fmt.Fprintf(&b, "%s\n", s.body)
	}
	if okPods == 0 {
		return nil, fmt.Errorf("scrape %s: all %d pod(s) failed to serve metrics; errors: %s",
			target.ComponentName, len(snaps), strings.Join(podErrors, "; "))
	}
	return parse(b.String(), queries), nil
}

// FindServicePort checks whether any Service in the given namespace whose selector
// contains appLabel=appValue exposes the specified targetPort (as either .spec.ports[].port
// or .spec.ports[].targetPort). Returns nil if found, or a descriptive error.
func FindServicePort(ctx context.Context, clientset kubernetes.Interface, namespace, appLabel, appValue string, targetPort int) error {
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing services in %s: %w", namespace, err)
	}
	for _, svc := range services.Items {
		if svc.Spec.Selector[appLabel] != appValue {
			continue
		}
		for _, p := range svc.Spec.Ports {
			if int(p.Port) == targetPort || p.TargetPort.IntValue() == targetPort {
				return nil
			}
		}
		var ports []string
		for _, p := range svc.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%s:%d->%s", p.Name, p.Port, p.TargetPort.String()))
		}
		return fmt.Errorf("service %s/%s selects %s=%s pods but does not expose port %d; declared ports: %v",
			namespace, svc.Name, appLabel, appValue, targetPort, ports)
	}
	return fmt.Errorf("no service in namespace %s selects %s=%s pods", namespace, appLabel, appValue)
}
