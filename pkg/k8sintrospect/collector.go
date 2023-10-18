package k8sintrospect

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/yaml"
)

const (
	logWindow = 20 * time.Minute

	maxLogLines        = 5000
	maxFirstLineCutOff = 1024    // only cut off first (partial line) if less than that many characters
	maxLogFileSize     = 1 << 20 // 1MB
)

var (
	log = logging.LoggerForModule()
)

type collector struct {
	ctx      context.Context
	callback FileCallback

	cfg Config

	client        kubernetes.Interface
	dynamicClient dynamic.Interface

	since       time.Time
	errors      []error
	shouldStuck bool
}

func newCollector(ctx context.Context, k8sRESTConfig *rest.Config, cfg Config, cb FileCallback, since time.Time, shouldStuck bool) (*collector, error) {
	restConfigShallowCopy := *k8sRESTConfig
	oldWrapTransport := restConfigShallowCopy.WrapTransport
	restConfigShallowCopy.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if oldWrapTransport != nil {
			rt = oldWrapTransport(rt)
		}
		return httputil.ContextBoundRoundTripper(ctx, rt)
	}

	k8sClient, err := kubernetes.NewForConfig(&restConfigShallowCopy)
	if err != nil {
		return nil, errors.Wrap(err, "could not create Kubernetes client set")
	}
	dynamicClient, err := dynamic.NewForConfig(&restConfigShallowCopy)
	if err != nil {
		return nil, errors.Wrap(err, "could not create dynamic Kubernetes client")
	}

	return &collector{
		ctx:           ctx,
		callback:      cb,
		cfg:           cfg,
		client:        k8sClient,
		dynamicClient: dynamicClient,
		since:         since,
		shouldStuck:   shouldStuck,
	}, nil
}

func generateFileName(obj k8sutil.Object, suffix string) string {
	namespace := obj.GetNamespace()
	if namespace == "" {
		namespace = "_global"
	}

	app := obj.GetLabels()["app"]
	if app == "" {
		app = obj.GetLabels()["app.kubernetes.io/name"]
	}
	if app == "" {
		app = "_ungrouped"
	}
	return fmt.Sprintf("%s/%s/%s-%s%s", namespace, app, strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName(), suffix)
}

func (c *collector) emitFile(obj k8sutil.Object, suffix string, data []byte) error {
	return c.emitFileRaw(generateFileName(obj, suffix), data)
}

func (c *collector) emitFileRaw(filePath string, data []byte) error {
	file := File{
		Path:     path.Join(c.cfg.PathPrefix, filePath),
		Contents: data,
	}

	return c.callback(c.ctx, file)
}

func (c *collector) createDynamicClients() map[schema.GroupVersionKind]dynamic.NamespaceableResourceInterface {
	gvkSet := make(map[schema.GroupVersionKind]struct{})
	for _, objCfg := range c.cfg.Objects {
		gvkSet[objCfg.GVK] = struct{}{}
	}

	_, apiResourceLists, err := c.client.Discovery().ServerGroupsAndResources()
	if err != nil {
		c.recordError(errors.Wrap(err, "failed to obtain server resources"))
		return nil
	}

	clientMap := make(map[schema.GroupVersionKind]dynamic.NamespaceableResourceInterface)
	for _, apiResourceList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			c.recordError(errors.Wrap(err, "failed to parse group/version for API resource list"))
			continue
		}
		for _, apiResource := range apiResourceList.APIResources {
			if strings.ContainsRune(apiResource.Name, '/') {
				continue
			}

			gvk := schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    apiResource.Kind,
			}
			if _, ok := gvkSet[gvk]; !ok {
				continue
			}
			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}
			clientMap[gvk] = c.dynamicClient.Resource(gvr)
		}
	}

	return clientMap
}

func (c *collector) collectPodData(pod *v1.Pod) error {
	objToMarshal := (interface{})(pod)

	var unstructuredPod unstructured.Unstructured
	if err := scheme.Scheme.Convert(pod, &unstructuredPod, nil); err == nil {
		RedactGeneric(&unstructuredPod)
		objToMarshal = &unstructuredPod
	}

	yamlData, err := yaml.Marshal(objToMarshal)
	if err != nil {
		yamlData = []byte(fmt.Sprintf("Error marshaling pod to YAML: %v", err))
	}
	if err := c.emitFile(pod, ".yaml", yamlData); err != nil {
		return err
	}

	sinceSeconds := int64(time.Since(c.since).Seconds())
	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Running != nil {
			podLogOpts := &v1.PodLogOptions{
				Container:    container.Name,
				SinceSeconds: &[]int64{sinceSeconds}[0],
				TailLines:    &[]int64{maxLogLines}[0],
			}
			logsData, err := c.client.CoreV1().Pods(pod.GetNamespace()).GetLogs(pod.GetName(), podLogOpts).DoRaw(c.ctx)
			if err != nil {
				logsData = []byte(fmt.Sprintf("Error retrieving container logs: %v\n", err))
				logsData = appendDebugError(logsData, err)
			} else {
				logsData = truncateLogData(logsData, maxLogFileSize, maxFirstLineCutOff)
			}

			if err := c.emitFile(pod, fmt.Sprintf("-logs-%s.txt", container.Name), logsData); err != nil {
				return err
			}
		}

		if container.LastTerminationState.Terminated != nil {
			sinceSeconds := metav1.NewTime(container.LastTerminationState.Terminated.FinishedAt.Add(-logWindow)).Unix()
			if (container.LastTerminationState.Terminated.StartedAt.Before(&metav1.Time{Time: c.since}) &&
				container.LastTerminationState.Terminated.FinishedAt.After(c.since)) {
				sinceSeconds = int64(time.Since(c.since).Seconds())
			}

			podLogOpts := &v1.PodLogOptions{
				Container:    container.Name,
				Previous:     true,
				SinceSeconds: &[]int64{sinceSeconds}[0],
				TailLines:    &[]int64{maxLogLines}[0],
			}
			logsData, err := c.client.CoreV1().Pods(pod.GetNamespace()).GetLogs(pod.GetName(), podLogOpts).DoRaw(c.ctx)
			if err != nil {
				logsData = []byte(fmt.Sprintf("Error retrieving previous container logs: %v\n", err))
				logsData = appendDebugError(logsData, err)
			} else {
				logsData = truncateLogData(logsData, maxLogFileSize, maxFirstLineCutOff)
			}

			if err := c.emitFile(pod, fmt.Sprintf("-logs-%s-previous.txt", container.Name), logsData); err != nil {
				return err
			}
		}
	}

	return nil
}

func appendDebugError(logsData []byte, err error) []byte {
	var serr *k8sErrors.StatusError
	if errors.As(err, &serr) {
		f, status := serr.DebugError()
		logsData = append(logsData, fmt.Sprintf(f, status)...)
	}
	return logsData
}

func (c *collector) recordError(err error) {
	if err != nil {
		c.errors = append(c.errors, err)
	}
}

func (c *collector) collectObjectsData(ns string, cfg ObjectConfig, resourceClient dynamic.NamespaceableResourceInterface) error {
	objList, err := resourceClient.Namespace(ns).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		c.recordError(err)
		return nil
	}

	for i := range objList.Items {
		obj := objList.Items[i]
		if cfg.FilterFunc != nil {
			if !cfg.FilterFunc(&obj) {
				continue
			}
		}

		RedactGeneric(&obj)
		if cfg.RedactionFunc != nil {
			cfg.RedactionFunc(&obj)
		}
		objYAML, err := yaml.Marshal(&obj)
		if err != nil {
			objYAML = []byte(fmt.Sprintf("Failed to marshal object to YAML: %v", err))
		}
		if err := c.emitFile(&obj, ".yaml", objYAML); err != nil {
			return err
		}
	}

	return nil
}

func (c *collector) collectNamespaceData(ns string) (bool, error) {
	namespace, err := c.client.CoreV1().Namespaces().Get(c.ctx, ns, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return false, nil
	}
	var nsYAML []byte
	if err == nil && namespace != nil {
		namespace.TypeMeta = metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		}
		nsYAML, err = yaml.Marshal(namespace)
	}

	if err != nil && len(nsYAML) == 0 {
		nsYAML = []byte(fmt.Sprintf("Failed to retrieve namespace: %v", err))
	}

	return true, c.emitFileRaw(fmt.Sprintf("%s/namespace-spec.yaml", ns), nsYAML)
}

func (c *collector) collectEventsData(ns string) error {
	eventList, err := c.client.CoreV1().Events(ns).List(c.ctx, metav1.ListOptions{
		Limit: 500,
	})
	if err != nil {
		return c.emitFileRaw(fmt.Sprintf("%s/event-list-error.txt", ns), []byte(err.Error()))
	}

	// Sort events, newest first
	events := eventList.Items
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.After(events[j].LastTimestamp.Time)
	})

	var csvContents bytes.Buffer
	csvWriter := csv.NewWriter(&csvContents)

	csvHeadings := []string{
		"Last Seen",
		"Frequency",
		"Type",
		"Reason",
		"Object",
		"Source",
		"Message",
	}

	if err := csvWriter.Write(csvHeadings); err != nil {
		return err
	}

	for _, event := range eventList.Items {
		var frequency string
		if event.Count > 1 {
			period := event.LastTimestamp.Sub(event.FirstTimestamp.Time)
			period = time.Duration(math.Round(float64(period)/float64(time.Minute)) * float64(time.Minute))
			frequency = fmt.Sprintf("%dx over %v", event.Count, period)
		}
		record := []string{
			event.LastTimestamp.Format(time.RFC3339),
			frequency,
			event.Type,
			event.Reason,
			fmt.Sprintf("%s/%s", strings.ToLower(event.InvolvedObject.Kind), event.Name),
			event.Source.Component,
			event.Message,
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	csvWriter.Flush()

	return c.emitFileRaw(fmt.Sprintf("%s/event-list.csv", ns), csvContents.Bytes())
}

func (c *collector) collectPodsData(ns string) error {
	podList, err := c.client.CoreV1().Pods(ns).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		c.recordError(errors.Wrapf(err, "could not list pods in namespace %q", ns))
		return nil
	}

	for i := range podList.Items {
		pod := podList.Items[i]
		pod.TypeMeta = metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		}

		if err := c.collectPodData(&pod); err != nil {
			return err
		}
	}

	return nil
}

func (c *collector) collectErrors() error {
	if len(c.errors) == 0 {
		return nil
	}

	var errorsText bytes.Buffer
	for _, err := range c.errors {
		fmt.Fprintln(&errorsText, err.Error())
	}

	return c.emitFileRaw("errors.txt", errorsText.Bytes())
}

// Run performs the collection process.
func (c *collector) Run() error {
	clientMap := c.createDynamicClients()

	for _, ns := range c.cfg.Namespaces {
		nsExists, err := c.collectNamespaceData(ns)
		if err != nil {
			return err
		}
		if !nsExists {
			continue
		}

		if err := c.collectPodsData(ns); err != nil {
			return err
		}
		for _, objCfg := range c.cfg.Objects {
			objClient := clientMap[objCfg.GVK]
			if objClient == nil {
				continue
			}
			if c.shouldStuck {
				time.Sleep(2 * time.Minute)
			}
			if err := c.collectObjectsData(ns, objCfg, objClient); err != nil {
				return err
			}
		}

		if err := c.collectEventsData(ns); err != nil {
			return err
		}
	}

	return c.collectErrors()
}
