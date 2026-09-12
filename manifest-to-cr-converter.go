package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"

	routev1 "github.com/openshift/api/route/v1"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

// ManifestConverter converts roxctl-generated manifests to Central CR
type ManifestConverter struct {
	centralCR       *platform.Central
	centralDeploy   *appsv1.Deployment
	scannerDeploy   *appsv1.Deployment
	centralDBDeploy *appsv1.Deployment
	centralService  *corev1.Service
	routes          map[string]*routev1.Route
	configMaps      map[string]*corev1.ConfigMap
	secrets         map[string]*corev1.Secret
	pvcs            map[string]*corev1.PersistentVolumeClaim
	decoder         runtime.Decoder
}

// NewManifestConverter creates a new converter instance
func NewManifestConverter() *ManifestConverter {
	// Create a YAML decoder that can handle Kubernetes resource quantities
	decoderCodecFactory := serializer.NewCodecFactory(scheme.Scheme)
	decoder := decoderCodecFactory.UniversalDeserializer()

	return &ManifestConverter{
		centralCR: &platform.Central{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "platform.stackrox.io/v1alpha1",
				Kind:       "Central",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stackrox-central-services",
				Namespace: "stackrox",
			},
			Spec: platform.CentralSpec{},
		},
		routes:     make(map[string]*routev1.Route),
		configMaps: make(map[string]*corev1.ConfigMap),
		secrets:    make(map[string]*corev1.Secret),
		pvcs:       make(map[string]*corev1.PersistentVolumeClaim),
		decoder:    decoder,
	}
}

// LoadManifests loads YAML manifests from files and directories
func (mc *ManifestConverter) LoadManifests(paths []string) error {
	if len(paths) == 0 || (len(paths) == 1 && paths[0] == "-") {
		return mc.loadFromReader(os.Stdin)
	}

	for _, path := range paths {
		// Check if path is a directory
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			if err := mc.loadFromDirectory(path); err != nil {
				return fmt.Errorf("failed to load directory %s: %w", path, err)
			}
			continue
		}

		// Load as regular file
		if err := mc.loadFromFile(path); err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}
	}
	return nil
}

// LoadFromNamespace loads resources from a Kubernetes namespace
func (mc *ManifestConverter) LoadFromNamespace(namespace string) error {
	return mc.loadFromNamespace(namespace)
}

func (mc *ManifestConverter) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return mc.loadFromReader(file)
}

func (mc *ManifestConverter) loadFromReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	var currentDoc strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "---") && currentDoc.Len() > 0 {
			if err := mc.parseDocument(currentDoc.String()); err != nil {
				return fmt.Errorf("failed to parse document: %w", err)
			}
			currentDoc.Reset()
		} else {
			currentDoc.WriteString(line + "\n")
		}
	}

	// Process the last document
	if currentDoc.Len() > 0 {
		if err := mc.parseDocument(currentDoc.String()); err != nil {
			return fmt.Errorf("failed to parse document: %w", err)
		}
	}

	return scanner.Err()
}

func (mc *ManifestConverter) loadFromDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		log.Printf("Loading manifest file: %s", path)
		return mc.loadFromFile(path)
	})
}

func (mc *ManifestConverter) loadFromNamespace(namespace string) error {
	log.Printf("Loading resources from Kubernetes namespace: %s", namespace)

	// Create Kubernetes client
	config, err := mc.getKubeConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	ctx := context.Background()

	// Fetch Deployments
	if err := mc.fetchDeployments(ctx, clientset, namespace); err != nil {
		return fmt.Errorf("failed to fetch deployments: %w", err)
	}

	// Fetch Services
	if err := mc.fetchServices(ctx, clientset, namespace); err != nil {
		return fmt.Errorf("failed to fetch services: %w", err)
	}

	// Fetch ConfigMaps
	if err := mc.fetchConfigMaps(ctx, clientset, namespace); err != nil {
		return fmt.Errorf("failed to fetch configmaps: %w", err)
	}

	// Fetch Secrets
	if err := mc.fetchSecrets(ctx, clientset, namespace); err != nil {
		return fmt.Errorf("failed to fetch secrets: %w", err)
	}

	// Fetch PVCs
	if err := mc.fetchPVCs(ctx, clientset, namespace); err != nil {
		return fmt.Errorf("failed to fetch pvcs: %w", err)
	}

	// Fetch Routes (OpenShift) - warns if API not available, fails on other errors
	if err := mc.fetchRoutes(ctx, config, namespace); err != nil {
		return fmt.Errorf("failed to fetch routes: %w", err)
	}

	return nil
}

func (mc *ManifestConverter) getKubeConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
}

func (mc *ManifestConverter) fetchDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, deploy := range deployments.Items {
		if deploy.ObjectMeta.Name == "central" {
			mc.centralDeploy = &deploy
		} else if deploy.ObjectMeta.Name == "scanner" {
			mc.scannerDeploy = &deploy
		}

	}

	return nil
}

func (mc *ManifestConverter) fetchServices(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, svc := range services.Items {
		if svc.ObjectMeta.Name == "central" {
			mc.centralService = &svc
		}

	}

	return nil
}

func (mc *ManifestConverter) fetchConfigMaps(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cm := range configMaps.Items {
		mc.configMaps[cm.ObjectMeta.Name] = &cm

	}

	return nil
}

func (mc *ManifestConverter) fetchSecrets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, secret := range secrets.Items {
		mc.secrets[secret.ObjectMeta.Name] = &secret

	}

	return nil
}

func (mc *ManifestConverter) fetchPVCs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pvc := range pvcs.Items {
		mc.pvcs[pvc.ObjectMeta.Name] = &pvc

	}

	return nil
}

func (mc *ManifestConverter) isRouteAPIAvailable(config *rest.Config) (bool, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, fmt.Errorf("failed to create discovery client: %w", err)
	}

	routeGV := schema.GroupVersion{Group: "route.openshift.io", Version: "v1"}
	apiResourceList, err := discoveryClient.ServerResourcesForGroupVersion(routeGV.String())
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, resource := range apiResourceList.APIResources {
		if resource.Name == "routes" {
			return true, nil
		}
	}

	return false, errors.New("route.openshift.io/v1 API found, but it lacks the routes resource")
}

func (mc *ManifestConverter) fetchRoutes(ctx context.Context, config *rest.Config, namespace string) error {
	routeAPIAvailable, err := mc.isRouteAPIAvailable(config)
	if err != nil {
		return fmt.Errorf("failed to check Route API availability: %w", err)
	}

	if !routeAPIAvailable {
		log.Printf("Warning: Route API (route.openshift.io/v1) not available - this is normal on non-OpenShift clusters")
		return nil
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client for routes: %w", err)
	}

	routeGVR := schema.GroupVersionResource{
		Group:    "route.openshift.io",
		Version:  "v1",
		Resource: "routes",
	}

	routes, err := dynamicClient.Resource(routeGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch routes from namespace %s: %w", namespace, err)
	}

	for _, unstructuredRoute := range routes.Items {
		yamlBytes, err := yaml.Marshal(&unstructuredRoute)
		if err != nil {
			return fmt.Errorf("failed to marshal route: %w", err)
		}

		obj, _, err := mc.decoder.Decode(yamlBytes, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to decode route: %w", err)
		}

		if route, ok := obj.(*routev1.Route); ok {
			mc.routes[route.ObjectMeta.Name] = route
		}
	}

	log.Printf("Fetched %d route(s) from namespace %s", len(routes.Items), namespace)
	return nil
}

func (mc *ManifestConverter) parseDocument(doc string) error {
	obj, _, err := mc.decoder.Decode([]byte(doc), nil, nil)
	if err != nil {
		return err
	}

	switch tObj := obj.(type) {
	case *appsv1.Deployment:
		if tObj.ObjectMeta.Name == "central" {
			mc.centralDeploy = tObj
		} else if tObj.ObjectMeta.Name == "scanner" {
			mc.scannerDeploy = tObj
		} else if tObj.ObjectMeta.Name == "central-db" {
			mc.centralDBDeploy = tObj
		} else {
			log.Printf("Ignoring unknown deployment object: %s", tObj.ObjectMeta.Name)
		}

	case *corev1.Service:
		if tObj.ObjectMeta.Name == "central" {
			mc.centralService = tObj
		} else {
			log.Printf("Ignoring unknown service object: %s", tObj.ObjectMeta.Name)
		}

	case *corev1.ConfigMap:
		mc.configMaps[tObj.ObjectMeta.Name] = tObj

	case *corev1.Secret:
		mc.secrets[tObj.ObjectMeta.Name] = tObj

	case *corev1.PersistentVolumeClaim:
		mc.pvcs[tObj.ObjectMeta.Name] = tObj

	case *routev1.Route:
		mc.routes[tObj.ObjectMeta.Name] = tObj
	default:
		log.Printf("Ignoring unknown document object of type %T", obj)
	}

	return nil
}

// Convert performs the conversion from manifests to Central CR
func (mc *ManifestConverter) Convert() error {
	mc.extractNamespace()

	if err := mc.extractCentralSettings(); err != nil {
		return fmt.Errorf("failed to extract central settings: %w", err)
	}

	if err := mc.extractScannerSettings(); err != nil {
		return fmt.Errorf("failed to extract scanner settings: %w", err)
	}

	if err := mc.extractExposureSettings(); err != nil {
		return fmt.Errorf("failed to extract exposure settings: %w", err)
	}

	if err := mc.extractDBSettings(); err != nil {
		return fmt.Errorf("failed to extract DB settings: %w", err)
	}

	if err := mc.extractTelemetrySettings(); err != nil {
		return fmt.Errorf("failed to extract telemetry settings: %w", err)
	}

	if err := mc.extractEgressSettings(); err != nil {
		return fmt.Errorf("failed to extract egress settings: %w", err)
	}

	return nil
}

func (mc *ManifestConverter) extractNamespace() {
	if mc.centralDeploy != nil && mc.centralDeploy.ObjectMeta.Namespace != "" {
		mc.centralCR.ObjectMeta.Namespace = mc.centralDeploy.ObjectMeta.Namespace
	}
}

func (mc *ManifestConverter) extractCentralSettings() error {
	if mc.centralDeploy == nil {
		return nil
	}

	if mc.centralCR.Spec.Central == nil {
		mc.centralCR.Spec.Central = &platform.CentralComponentSpec{}
	}

	mc.extractCentralResources()
	mc.extractCentralNodeSelector()
	mc.extractCentralTolerations()
	mc.extractCentralEnvironment()

	return nil
}

func (mc *ManifestConverter) extractCentralResources() {
	if mc.centralDeploy == nil || len(mc.centralDeploy.Spec.Template.Spec.Containers) == 0 {
		return
	}

	container := mc.centralDeploy.Spec.Template.Spec.Containers[0]
	if container.Resources.Requests != nil || container.Resources.Limits != nil {
		if mc.centralCR.Spec.Central.Resources == nil {
			mc.centralCR.Spec.Central.Resources = &corev1.ResourceRequirements{}
		}

		if container.Resources.Requests != nil {
			mc.centralCR.Spec.Central.Resources.Requests = container.Resources.Requests
		}
		if container.Resources.Limits != nil {
			mc.centralCR.Spec.Central.Resources.Limits = container.Resources.Limits
		}
	}
}

func (mc *ManifestConverter) extractCentralNodeSelector() {
	if mc.centralDeploy == nil {
		return
	}

	if mc.centralDeploy.Spec.Template.Spec.NodeSelector != nil {
		mc.centralCR.Spec.Central.NodeSelector = mc.centralDeploy.Spec.Template.Spec.NodeSelector
	}
}

func (mc *ManifestConverter) extractCentralTolerations() {
	if mc.centralDeploy == nil {
		return
	}

	if mc.centralDeploy.Spec.Template.Spec.Tolerations != nil {
		// Convert slice tolerations to slice of toleration pointers
		tolerations := make([]*corev1.Toleration, len(mc.centralDeploy.Spec.Template.Spec.Tolerations))
		for i, tol := range mc.centralDeploy.Spec.Template.Spec.Tolerations {
			tolerations[i] = &tol
		}
		mc.centralCR.Spec.Central.Tolerations = tolerations
	}
}

func (mc *ManifestConverter) extractCentralEnvironment() {
	if mc.centralDeploy == nil || len(mc.centralDeploy.Spec.Template.Spec.Containers) == 0 {
		return
	}

	container := mc.centralDeploy.Spec.Template.Spec.Containers[0]
	for _, env := range container.Env {
		switch env.Name {
		case "ROX_TELEMETRY_ENDPOINT":
			if mc.centralCR.Spec.Central.Telemetry == nil {
				mc.centralCR.Spec.Central.Telemetry = &platform.Telemetry{}
			}
			if mc.centralCR.Spec.Central.Telemetry.Storage == nil {
				mc.centralCR.Spec.Central.Telemetry.Storage = &platform.TelemetryStorage{}
			}
			mc.centralCR.Spec.Central.Telemetry.Storage.Endpoint = &env.Value

		case "ROX_TELEMETRY_STORAGE_KEY_V1":
			if mc.centralCR.Spec.Central.Telemetry == nil {
				mc.centralCR.Spec.Central.Telemetry = &platform.Telemetry{}
			}
			if mc.centralCR.Spec.Central.Telemetry.Storage == nil {
				mc.centralCR.Spec.Central.Telemetry.Storage = &platform.TelemetryStorage{}
			}
			mc.centralCR.Spec.Central.Telemetry.Storage.Key = &env.Value
		}
	}
}

func (mc *ManifestConverter) extractScannerSettings() error {
	if mc.scannerDeploy == nil {
		return nil
	}

	if mc.centralCR.Spec.Scanner == nil {
		mc.centralCR.Spec.Scanner = &platform.ScannerComponentSpec{}
	}

	// Extract analyzer replicas and resources
	if len(mc.scannerDeploy.Spec.Template.Spec.Containers) > 0 {
		if mc.centralCR.Spec.Scanner.Analyzer == nil {
			mc.centralCR.Spec.Scanner.Analyzer = &platform.ScannerAnalyzerComponent{}
		}

		// Extract replicas
		if mc.scannerDeploy.Spec.Replicas != nil {
			if mc.centralCR.Spec.Scanner.Analyzer.Scaling == nil {
				mc.centralCR.Spec.Scanner.Analyzer.Scaling = &platform.ScannerComponentScaling{}
			}
			mc.centralCR.Spec.Scanner.Analyzer.Scaling.Replicas = mc.scannerDeploy.Spec.Replicas
		}

		// Extract resources
		container := mc.scannerDeploy.Spec.Template.Spec.Containers[0]
		if container.Resources.Requests != nil || container.Resources.Limits != nil {
			if mc.centralCR.Spec.Scanner.Analyzer.Resources == nil {
				mc.centralCR.Spec.Scanner.Analyzer.Resources = &corev1.ResourceRequirements{}
			}

			if container.Resources.Requests != nil {
				mc.centralCR.Spec.Scanner.Analyzer.Resources.Requests = container.Resources.Requests
			}
			if container.Resources.Limits != nil {
				mc.centralCR.Spec.Scanner.Analyzer.Resources.Limits = container.Resources.Limits
			}
		}
	}

	return nil
}

func (mc *ManifestConverter) extractExposureSettings() error {
	// Always initialize Central and Exposure to support Route processing
	if mc.centralCR.Spec.Central == nil {
		mc.centralCR.Spec.Central = &platform.CentralComponentSpec{}
	}

	if mc.centralCR.Spec.Central.Exposure == nil {
		mc.centralCR.Spec.Central.Exposure = &platform.Exposure{}
	}

	if mc.centralService != nil {
		mc.extractServiceExposure()
	}

	mc.extractRouteExposure()

	return nil
}

func (mc *ManifestConverter) extractServiceExposure() {
	// Check service type to determine exposure method
	switch mc.centralService.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		if mc.centralCR.Spec.Central.Exposure.LoadBalancer == nil {
			mc.centralCR.Spec.Central.Exposure.LoadBalancer = &platform.ExposureLoadBalancer{}
		}
		mc.centralCR.Spec.Central.Exposure.LoadBalancer.Enabled = pointer.Bool(true)

		for _, port := range mc.centralService.Spec.Ports {
			if port.Name == "api" || port.Port == 443 {
				mc.centralCR.Spec.Central.Exposure.LoadBalancer.Port = &port.Port
				break
			}
		}

		if mc.centralService.Spec.LoadBalancerIP != "" {
			mc.centralCR.Spec.Central.Exposure.LoadBalancer.IP = &mc.centralService.Spec.LoadBalancerIP
		}

	case corev1.ServiceTypeNodePort:
		if mc.centralCR.Spec.Central.Exposure.NodePort == nil {
			mc.centralCR.Spec.Central.Exposure.NodePort = &platform.ExposureNodePort{}
		}
		mc.centralCR.Spec.Central.Exposure.NodePort.Enabled = pointer.Bool(true)

		for _, port := range mc.centralService.Spec.Ports {
			if port.Name == "api" || port.Port == 443 {
				if port.NodePort != 0 {
					mc.centralCR.Spec.Central.Exposure.NodePort.Port = &port.NodePort
				}
				break
			}
		}
	}
}

func (mc *ManifestConverter) extractRouteExposure() {
	centralRoute, ok := mc.routes["central"]
	if !ok {
		return
	}

	if mc.centralCR.Spec.Central.Exposure.Route == nil {
		mc.centralCR.Spec.Central.Exposure.Route = &platform.ExposureRoute{}
	}
	mc.centralCR.Spec.Central.Exposure.Route.Enabled = pointer.Bool(true)

	if centralRoute.Spec.Host != "" {
		mc.centralCR.Spec.Central.Exposure.Route.Host = &centralRoute.Spec.Host
	}
}

func (mc *ManifestConverter) extractDBSettings() error {
	if mc.centralDBDeploy == nil {
		return mc.extractExternalDBSettings()
	}

	if mc.centralCR.Spec.Central == nil {
		mc.centralCR.Spec.Central = &platform.CentralComponentSpec{}
	}

	if mc.centralCR.Spec.Central.DB == nil {
		mc.centralCR.Spec.Central.DB = &platform.CentralDBSpec{}
	}

	if len(mc.centralDBDeploy.Spec.Template.Spec.Containers) > 0 {
		container := mc.centralDBDeploy.Spec.Template.Spec.Containers[0]
		if container.Resources.Requests != nil || container.Resources.Limits != nil {
			if mc.centralCR.Spec.Central.DB.Resources == nil {
				mc.centralCR.Spec.Central.DB.Resources = &corev1.ResourceRequirements{}
			}

			if container.Resources.Requests != nil {
				mc.centralCR.Spec.Central.DB.Resources.Requests = container.Resources.Requests
			}
			if container.Resources.Limits != nil {
				mc.centralCR.Spec.Central.DB.Resources.Limits = container.Resources.Limits
			}
		}
	}

	mc.extractDBPersistence()

	return nil
}

func (mc *ManifestConverter) extractExternalDBSettings() error {
	for _, cm := range mc.configMaps {
		if strings.Contains(cm.ObjectMeta.Name, "central") {
			for key, value := range cm.Data {
				if strings.Contains(key, "config") && strings.Contains(value, "postgres://") {
					// Found external DB connection - extract connection string
					if mc.centralCR.Spec.Central == nil {
						mc.centralCR.Spec.Central = &platform.CentralComponentSpec{}
					}
					if mc.centralCR.Spec.Central.DB == nil {
						mc.centralCR.Spec.Central.DB = &platform.CentralDBSpec{}
					}

					connectionString := mc.extractConnectionString(value)
					if connectionString != "" {
						mc.centralCR.Spec.Central.DB.ConnectionStringOverride = &connectionString
					}
					return nil
				}
			}
		}
	}
	return nil
}

func (mc *ManifestConverter) extractConnectionString(config string) string {
	re := regexp.MustCompile(`postgres://[^\s"']+`)
	matches := re.FindStringSubmatch(config)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func (mc *ManifestConverter) extractDBPersistence() {
	var centralDBPVC *corev1.PersistentVolumeClaim
	for name, pvc := range mc.pvcs {
		if strings.Contains(name, "central-db") {
			centralDBPVC = pvc
			break
		}
	}

	if centralDBPVC == nil {
		return
	}

	if mc.centralCR.Spec.Central.DB.Persistence == nil {
		mc.centralCR.Spec.Central.DB.Persistence = &platform.DBPersistence{}
	}

	if mc.centralCR.Spec.Central.DB.Persistence.PersistentVolumeClaim == nil {
		mc.centralCR.Spec.Central.DB.Persistence.PersistentVolumeClaim = &platform.DBPersistentVolumeClaim{}
	}

	if centralDBPVC.ObjectMeta.Name != "" {
		mc.centralCR.Spec.Central.DB.Persistence.PersistentVolumeClaim.ClaimName = &centralDBPVC.ObjectMeta.Name
	}

	if centralDBPVC.Spec.StorageClassName != nil {
		mc.centralCR.Spec.Central.DB.Persistence.PersistentVolumeClaim.StorageClassName = centralDBPVC.Spec.StorageClassName
	}

	if storage, ok := centralDBPVC.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		sizeStr := storage.String()
		mc.centralCR.Spec.Central.DB.Persistence.PersistentVolumeClaim.Size = &sizeStr
	}
}

func (mc *ManifestConverter) extractTelemetrySettings() error {
	// Telemetry settings are primarily extracted from environment variables
	// which is handled in extractCentralEnvironment()

	// Check if telemetry is disabled by looking for specific keys
	for _, secret := range mc.secrets {
		for key, value := range secret.Data {
			if strings.Contains(key, "telemetry") && string(value) == "DISABLED" {
				if mc.centralCR.Spec.Central == nil {
					mc.centralCR.Spec.Central = &platform.CentralComponentSpec{}
				}
				if mc.centralCR.Spec.Central.Telemetry == nil {
					mc.centralCR.Spec.Central.Telemetry = &platform.Telemetry{}
				}
				mc.centralCR.Spec.Central.Telemetry.Enabled = pointer.Bool(false)
				return nil
			}
		}
	}

	// If we found telemetry endpoint/key in environment, enable it
	if mc.centralCR.Spec.Central != nil && mc.centralCR.Spec.Central.Telemetry != nil &&
		mc.centralCR.Spec.Central.Telemetry.Storage != nil &&
		(mc.centralCR.Spec.Central.Telemetry.Storage.Endpoint != nil ||
			mc.centralCR.Spec.Central.Telemetry.Storage.Key != nil) {
		mc.centralCR.Spec.Central.Telemetry.Enabled = pointer.Bool(true)
	}

	return nil
}

func (mc *ManifestConverter) extractEgressSettings() error {
	// Look for offline mode indicators
	if mc.centralDeploy != nil && len(mc.centralDeploy.Spec.Template.Spec.Containers) > 0 {
		container := mc.centralDeploy.Spec.Template.Spec.Containers[0]
		for _, env := range container.Env {
			if env.Name == "ROX_OFFLINE_MODE" {
				if mc.centralCR.Spec.Egress == nil {
					mc.centralCR.Spec.Egress = &platform.Egress{}
				}

				if env.Value == "true" {
					mc.centralCR.Spec.Egress.ConnectivityPolicy = platform.ConnectivityOffline.Pointer()
				} else {
					mc.centralCR.Spec.Egress.ConnectivityPolicy = platform.ConnectivityOnline.Pointer()
				}
				break
			}
		}
	}

	return nil
}

// OutputCR outputs the generated Central CR as YAML
func (mc *ManifestConverter) OutputCR() error {
	output, err := yaml.Marshal(mc.centralCR)
	if err != nil {
		return fmt.Errorf("failed to marshal Central CR: %w", err)
	}

	fmt.Print(string(output))
	return nil
}

func main() {
	var (
		namespace = flag.String("namespace", "", "Kubernetes namespace to fetch resources from")
		help      = flag.Bool("help", false, "Show help message")
	)

	// Custom usage function
	flag.Usage = func() {
		fmt.Printf(`Usage: %s [OPTIONS] [file1] [file2] [dir1] ...

Convert roxctl-generated manifests to Central CR YAML.

OPTIONS:
  --namespace string    Fetch resources from specified Kubernetes namespace
  --help               Show this help message

Input sources can be:
- YAML files: individual manifest files
- Directories: scanned recursively for .yaml/.yml files
- stdin: if no sources specified or '-' is used

Examples:
  # From files
  %s manifests.yaml
  %s file1.yaml file2.yaml

  # From directory
  %s ./manifests-dir/

  # From Kubernetes namespace
  %s --namespace stackrox

  # From stdin
  cat manifests.yaml | %s
  %s -

  # Mixed sources (files/directories + namespace)
  %s manifests.yaml ./more-manifests/ --namespace stackrox

For Kubernetes namespace access:
- Uses default kubeconfig (~/.kube/config) or KUBECONFIG environment variable
- Supports in-cluster configuration when running in a pod
- Fetches Deployments, Services, ConfigMaps, Secrets, and PVCs from the specified namespace

The program will analyze Kubernetes manifests generated by 'roxctl central generate'
and produce a Central custom resource that would generate similar manifests when
used with the StackRox operator.

Note: This is a best-effort conversion. Some settings may not be perfectly mapped,
and manual verification and adjustment may be required.
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	converter := NewManifestConverter()

	// Load from namespace if specified
	if *namespace != "" {
		if err := converter.LoadFromNamespace(*namespace); err != nil {
			log.Fatalf("Failed to load from namespace %s: %v", *namespace, err)
		}
	}

	// Load manifests from files/directories
	args := flag.Args()
	if len(args) > 0 {
		if err := converter.LoadManifests(args); err != nil {
			log.Fatalf("Failed to load manifests: %v", err)
		}
	} else if *namespace == "" {
		// No namespace and no file args, read from stdin
		if err := converter.LoadManifests([]string{}); err != nil {
			log.Fatalf("Failed to load manifests from stdin: %v", err)
		}
	}

	// Convert to Central CR
	if err := converter.Convert(); err != nil {
		log.Fatalf("Failed to convert manifests: %v", err)
	}

	// Output the result
	if err := converter.OutputCR(); err != nil {
		log.Fatalf("Failed to output Central CR: %v", err)
	}
}
