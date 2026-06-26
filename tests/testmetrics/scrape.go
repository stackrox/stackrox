package testmetrics

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// podSnapshot is a single /metrics (or custom path) scrape from one pod.
type podSnapshot struct {
	podName string
	body    string
	err     error
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
	podSubresourceName := podName
	if port > 0 {
		podSubresourceName = fmt.Sprintf("%s:%d", podName, port)
	}
	return clientset.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Resource("pods").
		Name(podSubresourceName).
		SubResource("proxy").
		Suffix(cleanMetricsPath(metricsPath)).
		Do(ctx).Raw()
}

// collectFromPods lists pods matching the target's selectors and scrapes metrics
// from each via the Kubernetes pods/proxy subresource.
func collectFromPods(ctx context.Context, clientset kubernetes.Interface, target ScrapeTarget) ([]podSnapshot, error) {
	if target.Namespace == "" {
		return nil, errors.New("testmetrics: collectFromPods: namespace is required")
	}
	list, err := clientset.CoreV1().Pods(target.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: target.LabelSelector,
		FieldSelector: target.FieldSelector,
	})
	if err != nil {
		return nil, err
	}
	out := make([]podSnapshot, 0, len(list.Items))
	for i := range list.Items {
		name := list.Items[i].Name
		body, err := scrapePodViaProxy(ctx, clientset, target.Namespace, name, target.MetricsPort, target.MetricsPath)
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

// ScrapeComponent scrapes pods of a single component and parses the collected metrics.
// Pods that fail to serve metrics are skipped; an error is returned only when no pod yields valid data.
func ScrapeComponent(ctx context.Context, clientset kubernetes.Interface, target ScrapeTarget) (Metrics, error) {
	snaps, err := collectFromPods(ctx, clientset, target)
	if err != nil {
		return Metrics{}, fmt.Errorf("scrape %s: %w", target.ComponentName, err)
	}
	if len(snaps) == 0 {
		return Metrics{}, fmt.Errorf("scrape %s: no pods found (selector=%q field=%q)",
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
		return Metrics{}, fmt.Errorf("scrape %s: all %d pod(s) failed to serve metrics; errors: %s",
			target.ComponentName, len(snaps), strings.Join(podErrors, "; "))
	}
	return Parse(b.String()), nil
}
