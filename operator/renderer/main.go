package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	centralReconciler "github.com/stackrox/rox/operator/internal/central/reconciler"
	securedclusterReconciler "github.com/stackrox/rox/operator/internal/securedcluster/reconciler"
	"github.com/stackrox/rox/pkg/version"
	rawZap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/yaml"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(platform.AddToScheme(scheme))
}

func main() {
	var centralCRPath string
	var securedclusterCRPath string
	var timeout time.Duration
	var verbose bool

	flag.StringVar(&centralCRPath, "central-cr", "", "Path to Central CR YAML file")
	flag.StringVar(&securedclusterCRPath, "securedcluster-cr", "", "Path to SecuredCluster CR YAML file")
	flag.DurationVar(&timeout, "timeout", 5*time.Minute, "Maximum time to wait for reconciliation")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	if centralCRPath == "" && securedclusterCRPath == "" {
		fmt.Fprintf(os.Stderr, "Error: at least one of --central-cr or --securedcluster-cr flags is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if err := run(centralCRPath, securedclusterCRPath, timeout, verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(centralCRPath, securedclusterCRPath string, timeout time.Duration, verbose bool) error {
	// Set up logging
	opts := zap.Options{
		Development: true,
	}
	if verbose {
		opts.Level = zapcore.DebugLevel
	}

	zapLogger := zap.NewRaw(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(zapr.NewLogger(zapLogger))
	logf.SetLogger(zapr.NewLogger(zapLogger))

	restore, err := rawZap.RedirectStdLogAt(zapLogger, zapcore.DebugLevel)
	if err != nil {
		return errors.Wrap(err, "unable to redirect std log")
	}
	defer restore()

	log := ctrl.Log.WithName("cr-renderer")
	log.Info("Starting CR renderer", "version", version.GetMainVersion())

	// Load CRs from files
	var centralCR *platform.Central
	var securedclusterCR *platform.SecuredCluster
	namespaces := make(map[string]struct{})

	if centralCRPath != "" {
		centralCR, err = loadCentralCR(centralCRPath)
		if err != nil {
			return errors.Wrap(err, "failed to load Central CR")
		}
		log.Info("Loaded Central CR", "name", centralCR.Name, "namespace", centralCR.Namespace)
		namespaces[centralCR.Namespace] = struct{}{}
	}

	if securedclusterCRPath != "" {
		securedclusterCR, err = loadSecuredClusterCR(securedclusterCRPath)
		if err != nil {
			return errors.Wrap(err, "failed to load SecuredCluster CR")
		}
		log.Info("Loaded SecuredCluster CR", "name", securedclusterCR.Name, "namespace", securedclusterCR.Namespace)
		namespaces[securedclusterCR.Namespace] = struct{}{}
	}

	// Set up envtest environment
	testEnv := &envtest.Environment{
		AttachControlPlaneOutput: verbose,
		CRDDirectoryPaths: []string{
			"config/crd/bases",
		},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start test environment")
	}
	defer func() {
		if err := testEnv.Stop(); err != nil {
			log.Error(err, "failed to stop test environment")
		}
	}()

	log.Info("Test environment started successfully")

	// set a var for the map-kube-apis helm plugin
	user, err := testEnv.AddUser(envtest.User{Name: "test", Groups: []string{"system:masters"}}, &rest.Config{})
	if err != nil {
		return errors.Wrap(err, "failed to add testenv user")
	}
	temp, err := os.CreateTemp(testEnv.ControlPlane.APIServer.CertDir, "*.kubecfg")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	defer func() {
		_ = temp.Close()
	}()
	contents, err := user.KubeConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get kube config")
	}
	_, err = temp.Write(contents)
	if err != nil {
		return errors.Wrap(err, "failed to write temp file")
	}
	os.Setenv("KUBECONFIG", temp.Name())

	// Create kubernetes client
	k8sClient, err := ctrlClient.New(cfg, ctrlClient.Options{Scheme: scheme})
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes client")
	}

	// Create controller manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0", // Disable metrics server
		},
		HealthProbeBindAddress: "0", // Disable health probe
		LeaderElection:         false,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create controller manager")
	}

	// Register reconcilers based on loaded CRs
	if centralCR != nil {
		if err := centralReconciler.RegisterNewReconciler(mgr, ""); err != nil {
			return errors.Wrap(err, "failed to register Central reconciler")
		}
		log.Info("Registered Central reconciler")
	}

	if securedclusterCR != nil {
		if err := securedclusterReconciler.RegisterNewReconciler(mgr, ""); err != nil {
			return errors.Wrap(err, "failed to register SecuredCluster reconciler")
		}
		log.Info("Registered SecuredCluster reconciler")
	}

	// Start the manager in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := mgr.Start(ctx); err != nil {
			log.Error(err, "failed to start controller manager")
		}
	}()

	// Wait for the manager to be ready
	log.Info("Waiting for controller manager to be ready...")
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		return errors.New("failed to sync cache")
	}

	log.Info("Controller manager is ready")

	// Capture baseline resources before applying the Central CR
	log.Info("Capturing baseline resources...")
	baselineResources, err := captureAllResources(ctx, cfg, "")
	if err != nil {
		return errors.Wrap(err, "failed to capture baseline resources")
	}
	log.Info("Captured baseline resources", "count", countResources(baselineResources))

	// Create namespaces if they don't exist
	for namespace := range namespaces {
		if err := createNamespaceIfNotExists(ctx, k8sClient, namespace); err != nil {
			return errors.Wrapf(err, "failed to create namespace %s", namespace)
		}
	}

	// Apply CRs
	var appliedCRs []ctrlClient.Object
	if centralCR != nil {
		log.Info("Applying Central CR", "name", centralCR.Name, "namespace", centralCR.Namespace)
		if err := k8sClient.Create(ctx, centralCR); err != nil {
			return errors.Wrap(err, "failed to create Central CR")
		}
		appliedCRs = append(appliedCRs, centralCR)
	}

	if securedclusterCR != nil {
		log.Info("Applying SecuredCluster CR", "name", securedclusterCR.Name, "namespace", securedclusterCR.Namespace)
		if err := k8sClient.Create(ctx, securedclusterCR); err != nil {
			return errors.Wrap(err, "failed to create SecuredCluster CR")
		}
		appliedCRs = append(appliedCRs, securedclusterCR)
	}

	// Wait for reconciliation to complete by watching the status
	log.Info("Waiting for reconciliation to complete...")
	if err := waitForAllReconciliation(ctx, k8sClient, appliedCRs, timeout); err != nil {
		return errors.Wrap(err, "reconciliation failed or timed out")
	}

	log.Info("Reconciliation completed successfully")

	// Capture all resources after reconciliation
	log.Info("Capturing final resources...")
	finalResources, err := captureAllResources(ctx, cfg, "")
	if err != nil {
		return errors.Wrap(err, "failed to capture final resources")
	}
	log.Info("Captured final resources", "count", countResources(finalResources))

	// Filter out baseline resources to get only reconciler-created resources
	log.Info("Filtering reconciler-created resources...")
	reconciledResources := filterResources(finalResources, baselineResources)
	reconciledCount := countResources(reconciledResources)
	log.Info("Found reconciler-created resources", "count", reconciledCount)

	// Write reconciler-created resources to YAML files
	log.Info("Writing reconciler-created resources to files...")
	if err := writeResourcesToFiles(reconciledResources, "reconciler-output"); err != nil {
		return errors.Wrap(err, "failed to write resources to files")
	}

	log.Info("Resources written to reconciler-output/ directory")

	return nil
}

func loadCentralCR(path string) (*platform.Central, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read CR file")
	}

	var central platform.Central
	if err := yaml.Unmarshal(data, &central); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal CR YAML")
	}

	// Set default namespace if not specified
	if central.Namespace == "" {
		central.Namespace = "stackrox"
	}

	return &central, nil
}

func loadSecuredClusterCR(path string) (*platform.SecuredCluster, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read CR file")
	}

	var securedcluster platform.SecuredCluster
	if err := yaml.Unmarshal(data, &securedcluster); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal CR YAML")
	}

	// Set default namespace if not specified
	if securedcluster.Namespace == "" {
		securedcluster.Namespace = "stackrox"
	}

	return &securedcluster, nil
}

func createNamespaceIfNotExists(ctx context.Context, client ctrlClient.Client, namespace string) error {
	ns := &coreV1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	err := client.Create(ctx, ns)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func waitForAllReconciliation(ctx context.Context, client ctrlClient.Client, crs []ctrlClient.Object, timeout time.Duration) error {
	log := ctrl.Log.WithName("wait-reconciliation")

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Info("Polling for CR status changes...", "count", len(crs))

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return errors.New("timeout waiting for reconciliation")
		case <-ticker.C:
			allComplete := true

			for _, cr := range crs {
				complete, err := checkCRReconciliation(ctx, client, cr, log)
				if err != nil {
					return err
				}
				if !complete {
					allComplete = false
				}
			}

			if allComplete {
				log.Info("All reconciliations completed successfully")
				return nil
			}
		}
	}
}

func checkCRReconciliation(ctx context.Context, client ctrlClient.Client, cr ctrlClient.Object, log logr.Logger) (bool, error) {
	switch typedCR := cr.(type) {
	case *platform.Central:
		currentCentral := &platform.Central{}
		err := client.Get(ctx, ctrlClient.ObjectKey{
			Name:      typedCR.Name,
			Namespace: typedCR.Namespace,
		}, currentCentral)
		if err != nil {
			log.V(1).Info("Failed to get Central CR", "error", err)
			return false, nil
		}

		log.V(1).Info("Central CR status update",
			"name", currentCentral.Name,
			"conditions", len(currentCentral.Status.Conditions),
			"productVersion", currentCentral.Status.ProductVersion)

		if err := errorCondition(currentCentral.Status.Conditions); err != nil {
			return false, fmt.Errorf("Central reconciliation failed with error condition: %w", err)
		}

		return isReconciliationComplete(currentCentral), nil

	case *platform.SecuredCluster:
		currentSecuredCluster := &platform.SecuredCluster{}
		err := client.Get(ctx, ctrlClient.ObjectKey{
			Name:      typedCR.Name,
			Namespace: typedCR.Namespace,
		}, currentSecuredCluster)
		if err != nil {
			log.V(1).Info("Failed to get SecuredCluster CR", "error", err)
			return false, nil
		}

		log.V(1).Info("SecuredCluster CR status update",
			"name", currentSecuredCluster.Name,
			"conditions", len(currentSecuredCluster.Status.Conditions))

		if err := errorCondition(currentSecuredCluster.Status.Conditions); err != nil {
			return false, fmt.Errorf("SecuredCluster reconciliation failed with error condition: %w", err)
		}

		return isSecuredClusterReconciliationComplete(currentSecuredCluster), nil

	default:
		return false, errors.Errorf("unsupported CR type: %T", cr)
	}
}

func isReconciliationComplete(central *platform.Central) bool {
	for _, condition := range central.Status.Conditions {
		if condition.Type == platform.ConditionDeployed && condition.Status == platform.StatusTrue {
			return true
		}
	}

	return false
}

func isSecuredClusterReconciliationComplete(securedcluster *platform.SecuredCluster) bool {
	for _, condition := range securedcluster.Status.Conditions {
		if condition.Type == platform.ConditionDeployed && condition.Status == platform.StatusTrue {
			return true
		}
	}
	return false
}

func errorCondition(conditions []platform.StackRoxCondition) error {
	for _, condition := range conditions {
		if condition.Type == platform.ConditionDeployed && condition.Status == platform.StatusFalse {
			return errors.New(string(condition.Message))
		}
	}
	return nil
}

func listResourcesOfType(ctx context.Context, client ctrlClient.Client, gvk schema.GroupVersionKind, namespace string, namespaced bool) ([]unstructured.Unstructured, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	})

	var listOpts []ctrlClient.ListOption
	if namespaced && namespace != "" {
		listOpts = append(listOpts, ctrlClient.InNamespace(namespace))
	}

	if err := client.List(ctx, list, listOpts...); err != nil {
		return nil, err
	}

	return list.Items, nil
}

func shouldSkipResource(gvk schema.GroupVersionKind) bool {
	// Skip some noisy system resources
	skipGroups := []string{
		"metrics.k8s.io",
		"coordination.k8s.io",
		"node.k8s.io",
		"flowcontrol.apiserver.k8s.io",
		"apiregistration.k8s.io",
	}

	for _, skipGroup := range skipGroups {
		if gvk.Group == skipGroup {
			return true
		}
	}

	// Skip some specific resources
	skipKinds := []string{
		"Event",
		"Lease",
		"EndpointSlice",
		"Node",
		"ComponentStatus",
	}

	for _, skipKind := range skipKinds {
		if gvk.Kind == skipKind {
			return true
		}
	}

	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func captureAllResources(ctx context.Context, cfg *rest.Config, namespace string) (map[schema.GroupVersionKind][]unstructured.Unstructured, error) {
	log := ctrl.Log.WithName("resource-capture")

	// Create a discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create discovery client")
	}

	// Create a kubernetes client for listing resources
	k8sClient, err := ctrlClient.New(cfg, ctrlClient.Options{Scheme: scheme})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes client")
	}

	// Get all API resources
	_, apiResourceLists, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return nil, errors.Wrap(err, "failed to discover API resources")
	}

	resources := make(map[schema.GroupVersionKind][]unstructured.Unstructured)

	for _, apiResourceList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, resource := range apiResourceList.APIResources {
			if !contains(resource.Verbs, "list") {
				continue // Skip resources that don't support listing
			}

			gvk := schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    resource.Kind,
			}

			// Skip some system resources to reduce noise
			if shouldSkipResource(gvk) {
				continue
			}

			resourceList, err := listResourcesOfType(ctx, k8sClient, gvk, namespace, resource.Namespaced)
			if err != nil {
				log.V(1).Info("Failed to list resources", "gvk", gvk, "error", err)
				continue
			}

			if len(resourceList) > 0 {
				resources[gvk] = resourceList
			}
		}
	}

	return resources, nil
}

func filterResources(finalResources, baselineResources map[schema.GroupVersionKind][]unstructured.Unstructured) map[schema.GroupVersionKind][]unstructured.Unstructured {
	filtered := make(map[schema.GroupVersionKind][]unstructured.Unstructured)

	for gvk, finalList := range finalResources {
		baselineList := baselineResources[gvk]
		baselineMap := make(map[string]bool)

		// Create a map of baseline resource UIDs for fast lookup
		for _, baselineRes := range baselineList {
			uid := string(baselineRes.GetUID())
			if uid != "" {
				baselineMap[uid] = true
			}
		}

		// Filter out resources that existed in baseline
		var newResources []unstructured.Unstructured
		for _, finalRes := range finalList {
			uid := string(finalRes.GetUID())
			if uid != "" && !baselineMap[uid] {
				newResources = append(newResources, finalRes)
			}
		}

		if len(newResources) > 0 {
			filtered[gvk] = newResources
		}
	}

	return filtered
}

func countResources(resources map[schema.GroupVersionKind][]unstructured.Unstructured) int {
	count := 0
	for _, list := range resources {
		count += len(list)
	}
	return count
}

func writeResourcesToFiles(resources map[schema.GroupVersionKind][]unstructured.Unstructured, outputDir string) error {
	log := ctrl.Log.WithName("write-resources")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create output directory")
	}

	for gvk, resourceList := range resources {
		// Generate filename: group_version_kind.yaml
		// Handle empty group (core API)
		group := gvk.Group
		if group == "" {
			group = "core"
		}
		filename := fmt.Sprintf("%s_%s_%s.yaml", group, gvk.Version, strings.ToLower(gvk.Kind))
		// Replace problematic characters in filename
		filename = strings.ReplaceAll(filename, "/", "_")
		filename = strings.ReplaceAll(filename, ":", "_")

		filepath := fmt.Sprintf("%s/%s", outputDir, filename)

		log.Info("Writing resources to file", "gvk", gvk, "count", len(resourceList), "file", filepath)

		var yamlDocs []string
		for _, resource := range resourceList {
			// Strip managedFields from metadata to clean up the output
			resourceCopy := resource.DeepCopy()
			if metadata, exists := resourceCopy.Object["metadata"]; exists {
				if metadataMap, ok := metadata.(map[string]interface{}); ok {
					delete(metadataMap, "managedFields")
				}
			}

			// Convert unstructured to YAML
			yamlBytes, err := yaml.Marshal(resourceCopy.Object)
			if err != nil {
				log.Error(err, "Failed to marshal resource to YAML", "resource", resource.GetName())
				continue
			}
			yamlDocs = append(yamlDocs, string(yamlBytes))
		}

		if len(yamlDocs) > 0 {
			// Join all resources with YAML document separator
			content := strings.Join(yamlDocs, "---\n")

			if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
				return errors.Wrapf(err, "failed to write file %s", filepath)
			}
		}
	}

	return nil
}
