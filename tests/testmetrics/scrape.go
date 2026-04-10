package testmetrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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

// PodSnapshot is a single /metrics (or custom path) scrape from one pod.
type PodSnapshot struct {
	PodName string
	Body    string
	Err     error
}

// PodScrapeOptions configures ScrapePodViaProxy (Kubernetes pods/proxy subresource).
type PodScrapeOptions struct {
	Namespace   string
	PodName     string
	Port        int
	MetricsPath string
}

// PodCollectOptions scrapes metrics from every pod matching LabelSelector.
type PodCollectOptions struct {
	Namespace     string
	LabelSelector string
	FieldSelector string
	Port          int
	MetricsPath   string
	Transport     Transport
	RestConfig    *rest.Config
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

// ScrapePodViaProxy scrapes Prometheus text from a pod via GET .../pods/{name}[:port]/proxy/{path}.
func ScrapePodViaProxy(ctx context.Context, clientset kubernetes.Interface, opts PodScrapeOptions) ([]byte, error) {
	if opts.Namespace == "" || opts.PodName == "" {
		return nil, errors.New("testmetrics: ScrapePodViaProxy: namespace and pod name are required")
	}
	path := strings.Trim(opts.MetricsPath, "/")
	if path == "" {
		path = "metrics"
	}
	segments := strings.Split(path, "/")
	podSubresourceName := opts.PodName
	if opts.Port > 0 {
		podSubresourceName = fmt.Sprintf("%s:%d", opts.PodName, opts.Port)
	}
	req := clientset.CoreV1().RESTClient().Get().
		Namespace(opts.Namespace).
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

// ScrapePodViaPortForward opens an ephemeral port-forward to remotePort on the pod
// and GETs metricsPath on localhost.
func ScrapePodViaPortForward(ctx context.Context, restCfg *rest.Config, namespace, podName string, remotePort int, metricsPath string) ([]byte, error) {
	if restCfg == nil {
		return nil, errors.New("testmetrics: ScrapePodViaPortForward: rest.Config is required")
	}
	if namespace == "" || podName == "" {
		return nil, errors.New("testmetrics: ScrapePodViaPortForward: namespace and pod name are required")
	}
	if remotePort <= 0 {
		return nil, errors.New("testmetrics: ScrapePodViaPortForward: remote metrics port must be positive")
	}
	path := metricsPath
	if path == "" {
		path = "/metrics"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	restClient, err := rest.RESTClientFor(rest.CopyConfig(restCfg))
	if err != nil {
		return nil, fmt.Errorf("testmetrics: port-forward: %w", err)
	}
	req := restClient.Post().Resource("pods").Namespace(namespace).Name(podName).SubResource("portforward")
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
	resp, err := http.DefaultClient.Do(httpReq)
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

func scrapePod(ctx context.Context, clientset kubernetes.Interface, opts PodCollectOptions, podName string) ([]byte, error) {
	transport := opts.Transport
	if transport == "" {
		transport = TransportProxy
	}
	path := opts.MetricsPath
	if path == "" {
		path = "metrics"
	}
	switch transport {
	case TransportPortForward:
		if opts.RestConfig == nil {
			return nil, errors.New("testmetrics: PodCollectOptions.RestConfig is required for port-forward transport")
		}
		if opts.Port <= 0 {
			return nil, errors.New("testmetrics: PodCollectOptions.Port must be set for port-forward transport")
		}
		return ScrapePodViaPortForward(ctx, opts.RestConfig, opts.Namespace, podName, opts.Port, path)
	case TransportProxy:
		return ScrapePodViaProxy(ctx, clientset, PodScrapeOptions{
			Namespace:   opts.Namespace,
			PodName:     podName,
			Port:        opts.Port,
			MetricsPath: path,
		})
	default:
		return nil, fmt.Errorf("testmetrics: unsupported metrics transport %q", transport)
	}
}

// CollectFromPods lists pods matching LabelSelector/FieldSelector and scrapes metrics
// from each (same port/path/transport).
func CollectFromPods(ctx context.Context, clientset kubernetes.Interface, opts PodCollectOptions) ([]PodSnapshot, error) {
	if opts.Namespace == "" {
		return nil, errors.New("testmetrics: CollectFromPods: namespace is required")
	}
	if opts.Transport == TransportPortForward && opts.RestConfig == nil {
		return nil, errors.New("testmetrics: CollectFromPods: RestConfig required for port-forward")
	}
	list, err := clientset.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
	})
	if err != nil {
		return nil, err
	}
	out := make([]PodSnapshot, 0, len(list.Items))
	for i := range list.Items {
		name := list.Items[i].Name
		body, err := scrapePod(ctx, clientset, opts, name)
		snap := PodSnapshot{PodName: name}
		if err != nil {
			snap.Err = err
		} else {
			snap.Body = string(body)
		}
		out = append(out, snap)
	}
	return out, nil
}

// ScrapeComponent scrapes pods of a single component and parses the requested counters.
// Pods that fail to serve metrics are skipped; an error is returned only when no pod yields valid data.
func ScrapeComponent(ctx context.Context, clientset kubernetes.Interface, target ScrapeTarget, transport Transport, queries []Query) (map[string]Value, error) {
	snaps, err := CollectFromPods(ctx, clientset, PodCollectOptions{
		Namespace:     target.Namespace,
		LabelSelector: target.LabelSelector,
		FieldSelector: target.FieldSelector,
		Port:          target.MetricsPort,
		MetricsPath:   target.MetricsPath,
		Transport:     transport,
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
	for _, s := range snaps {
		if s.Err != nil {
			continue
		}
		okPods++
		fmt.Fprintf(&b, "%s\n", s.Body)
	}
	if okPods == 0 {
		return nil, fmt.Errorf("scrape %s: all %d pod(s) failed to serve metrics", target.ComponentName, len(snaps))
	}
	return Parse(b.String(), queries), nil
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
