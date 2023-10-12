package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	clusterUtil "github.com/stackrox/rox/central/cluster/util"
	centralDetection "github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
	deploytimePkg "github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifier"
	resourcesConv "github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	k8sSerializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Detection)): {
			"/v1.DetectionService/DetectBuildTime",
			"/v1.DetectionService/DetectDeployTimeFromYAML",
		},
		or.SensorOr(user.With(permissions.Modify(resources.Detection))): {
			"/v1.DetectionService/DetectDeployTime",
		},
	})

	log = logging.LoggerForModule()

	workloadScheme = k8sRuntime.NewScheme()

	workloadDeserializer = k8sSerializer.NewCodecFactory(workloadScheme).UniversalDeserializer()

	delegateScanPermissions = []string{"Image"}
)

func init() {
	metav1.AddToGroupVersion(workloadScheme, k8sSchema.GroupVersion{Version: "v1"})
	pkgUtils.Must(errors.Wrap(scheme.AddToScheme(workloadScheme), "failed to load scheme"))
	pkgUtils.Must(errors.Wrap(k8sutil.AddOpenShiftSchemes(workloadScheme), "failed to load openshift schemes"))
}

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	apiV1.UnimplementedDetectionServiceServer

	policySet          detection.PolicySet
	imageEnricher      enricher.ImageEnricher
	imageDatastore     imageDatastore.DataStore
	riskManager        manager.Manager
	deploymentEnricher enrichment.Enricher
	buildTimeDetector  buildtime.Detector
	clusters           clusterDatastore.DataStore

	notifications notifier.Processor

	detector deploytime.Detector

	clusterSACHelper sachelper.ClusterSacHelper
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	apiV1.RegisterDetectionServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return apiV1.RegisterDetectionServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) maybeSendNotifications(req *apiV1.BuildDetectionRequest, alerts []*storage.Alert) {
	if !req.GetSendNotifications() {
		return
	}
	for _, alert := range alerts {
		// We use context.Background() instead of the request context because it is possible (and expected) that the
		// sending of notifications will take place asynchronously, and will still be happening after the request is done.
		s.notifications.ProcessAlert(context.Background(), alert)
	}
}

// DetectBuildTime runs detection on a built image.
func (s *serviceImpl) DetectBuildTime(ctx context.Context, req *apiV1.BuildDetectionRequest) (*apiV1.BuildDetectionResponse, error) {
	image := req.GetImage()
	if req.GetImageName() != "" {
		var err error
		image, err = utils.GenerateImageFromString(req.GetImageName())
		if err != nil {
			return nil, err
		}
	}
	if image.GetName() == nil {
		return nil, errox.InvalidArgs.CausedBy("image or image_name must be specified")
	}
	// This is a workaround for those who post the full image, but don't fill in fullname
	if name := image.GetName(); name != nil && name.GetFullName() == "" {
		name.FullName = types.Wrapper{GenericImage: image}.FullName()
	}

	img := types.ToImage(image)

	enrichmentContext := enricher.EnrichmentContext{}
	fetchOpt, err := getFetchOptionFromRequest(req)
	if err != nil {
		return nil, err
	}

	if req.GetCluster() != "" {
		// The request indicates enrichment should be delegated to a specific cluster.
		clusterID, err := clusterUtil.GetClusterIDFromNameOrID(ctx, s.clusterSACHelper, req.GetCluster(), delegateScanPermissions)
		if err != nil {
			return nil, err
		}

		enrichmentContext.ClusterID = clusterID
	}

	enrichmentContext.FetchOpt = fetchOpt
	enrichmentContext.Delegable = true
	enrichResult, err := s.imageEnricher.EnrichImage(ctx, enrichmentContext, img)
	if err != nil {
		return nil, err
	}
	if enrichResult.ImageUpdated {
		img.Id = utils.GetSHA(img)
		if img.GetId() != "" {
			if err := s.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
				return nil, err
			}
		}
	}
	utils.FilterSuppressedCVEsNoClone(img)
	filter, getUnusedCategories := centralDetection.MakeCategoryFilter(req.GetPolicyCategories())
	alerts, err := s.buildTimeDetector.Detect(img, filter)
	if err != nil {
		return nil, err
	}
	unusedCategories := getUnusedCategories()
	if len(unusedCategories) > 0 {
		return nil, fmt.Errorf("allowed categories %q did not match any policy categories", unusedCategories)
	}

	s.maybeSendNotifications(req, alerts)

	return &apiV1.BuildDetectionResponse{
		Alerts: alerts,
	}, nil
}

func (s *serviceImpl) enrichAndDetect(ctx context.Context, enrichmentContext enricher.EnrichmentContext, deployment *storage.Deployment, policyCategories ...string) (*apiV1.DeployDetectionResponse_Run, error) {
	images, updatedIndices, _, err := s.deploymentEnricher.EnrichDeployment(ctx, enrichmentContext, deployment)
	if err != nil {
		return nil, err
	}
	for _, idx := range updatedIndices {
		img := images[idx]
		img.Id = utils.GetSHA(img)
		if err := s.riskManager.CalculateRiskAndUpsertImage(images[idx]); err != nil {
			return nil, err
		}
	}
	for _, img := range images {
		utils.FilterSuppressedCVEsNoClone(img)
	}

	detectionCtx := deploytimePkg.DetectionContext{
		EnforcementOnly: enrichmentContext.EnforcementOnly,
	}

	filter, getUnusedCategories := centralDetection.MakeCategoryFilter(policyCategories)
	alerts, err := s.detector.Detect(detectionCtx, booleanpolicy.EnhancedDeployment{
		Deployment: deployment,
		Images:     images,
	}, filter)
	if err != nil {
		return nil, err
	}
	unusedCategories := getUnusedCategories()
	if len(unusedCategories) > 0 {
		return nil, errors.Errorf("allowed categories %v did not match any policy categories", unusedCategories)
	}
	return &apiV1.DeployDetectionResponse_Run{
		Name:   deployment.GetName(),
		Type:   deployment.GetType(),
		Alerts: alerts,
	}, nil
}

func (s *serviceImpl) runDeployTimeDetect(ctx context.Context, enrichmentContext enricher.EnrichmentContext, obj k8sRuntime.Object, policyCategories []string) (*apiV1.DeployDetectionResponse_Run, error) {
	if !kubernetes.IsDeploymentResource(obj.GetObjectKind().GroupVersionKind().Kind) {
		return nil, nil
	}

	deployment, err := resourcesConv.NewDeploymentFromStaticResource(obj, obj.GetObjectKind().GroupVersionKind().Kind, "", "")
	if err != nil {
		return nil, errox.InvalidArgs.New("could not convert to deployment from resource").CausedBy(err)
	}
	return s.enrichAndDetect(ctx, enrichmentContext, deployment, policyCategories...)
}

func getObjectsFromYAML(yamlString string) (objects []k8sRuntime.Object, ignoredObjectRefs []string, err error) {
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewBufferString(yamlString)))
	for {
		yamlBytes, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil,
				errox.InvalidArgs.New("failed to read yaml").CausedBy(err)
		}
		obj, _, err := workloadDeserializer.Decode(yamlBytes, nil, nil)
		if err != nil {
			// Only return errors if the resource's schema is not registered.
			if !k8sRuntime.IsNotRegisteredError(err) {
				return nil, nil,
					errox.InvalidArgs.New("could not parse yaml").CausedBy(err)
			}
			// Save the ignored object, so we can return it to the caller and skip it.
			ignoredObj, err := getIgnoredObjectRefFromYAML(string(yamlBytes))
			if err != nil {
				return nil, nil,
					errox.InvariantViolation.New("could not get ignored object").CausedBy(err)
			}
			ignoredObjectRefs = append(ignoredObjectRefs, ignoredObj)
			continue
		}

		if list, ok := obj.(*coreV1.List); ok {
			listResources, ignoredObjs, err := getObjectsFromList(list)
			if err != nil {
				return nil, nil, err
			}
			objects = append(objects, listResources...)
			ignoredObjectRefs = append(ignoredObjectRefs, ignoredObjs...)
		} else {
			objects = append(objects, obj)
		}
	}
	return objects, ignoredObjectRefs, nil
}

func getObjectsFromList(list *coreV1.List) ([]k8sRuntime.Object, []string, error) {
	objects := make([]k8sRuntime.Object, 0, len(list.Items))
	var ignoredObjectsRefs []string
	for i, item := range list.Items {
		obj, _, err := workloadDeserializer.Decode(item.Raw, nil, nil)
		if err == nil {
			objects = append(objects, obj)
			continue
		}

		if !k8sRuntime.IsNotRegisteredError(err) {
			return nil, nil,
				errox.InvalidArgs.Newf("could not decode item %d in the list", i).CausedBy(err)
		}
		ignoredObjRef, err := getIgnoredObjectRefFromYAML(string(item.Raw))
		if err != nil {
			return nil, nil, errox.InvariantViolation.New("could not get ignored object").CausedBy(err)
		}
		ignoredObjectsRefs = append(ignoredObjectsRefs, ignoredObjRef)
	}
	return objects, ignoredObjectsRefs, nil
}

// DetectDeployTimeFromYAML runs detection on a deployment.
func (s *serviceImpl) DetectDeployTimeFromYAML(ctx context.Context, req *apiV1.DeployYAMLDetectionRequest) (*apiV1.DeployDetectionResponse, error) {
	if req.GetYaml() == "" {
		return nil, errox.InvalidArgs.CausedBy("yaml field must be specified in detection request")
	}

	resources, ignoredObjectRefs, err := getObjectsFromYAML(req.GetYaml())
	if err != nil {
		return nil, err
	}

	eCtx := enricher.EnrichmentContext{
		EnforcementOnly: req.GetEnforcementOnly(),
		Delegable:       true,
	}
	fetchOpt, err := getFetchOptionFromRequest(req)
	if err != nil {
		return nil, err
	}
	eCtx.FetchOpt = fetchOpt

	if req.GetCluster() != "" {
		// The request indicates enrichment should be delegated to a specific cluster.
		clusterID, err := clusterUtil.GetClusterIDFromNameOrID(ctx, s.clusterSACHelper, req.GetCluster(), delegateScanPermissions)
		if err != nil {
			return nil, err
		}

		eCtx.ClusterID = clusterID
	}

	var runs []*apiV1.DeployDetectionResponse_Run
	for _, r := range resources {
		run, err := s.runDeployTimeDetect(ctx, eCtx, r, req.GetPolicyCategories())
		if err != nil {
			return nil, errox.InvalidArgs.New("unable to convert object").CausedBy(err)
		}
		if run != nil {
			runs = append(runs, run)
		}
	}
	return &apiV1.DeployDetectionResponse{
		Runs:              runs,
		IgnoredObjectRefs: ignoredObjectRefs,
	}, nil
}

func isDeployTimeEnforcement(actions []storage.EnforcementAction) bool {
	for _, a := range actions {
		if a == storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT || a == storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT {
			return true
		}
	}
	return false
}

func (s *serviceImpl) populateDeploymentWithClusterInfo(ctx context.Context, clusterID string, deployment *storage.Deployment) error {
	if clusterID == "" {
		return nil
	}
	clusterName, exists, err := s.clusters.GetClusterName(ctx, clusterID)
	if err != nil {
		return err
	}
	if !exists {
		return errox.InvalidArgs.Newf("cluster with ID %q does not exist", clusterID)
	}
	deployment.ClusterId = clusterID
	deployment.ClusterName = clusterName
	return nil
}

// DetectDeployTime runs detection on a deployment.
func (s *serviceImpl) DetectDeployTime(ctx context.Context, req *apiV1.DeployDetectionRequest) (*apiV1.DeployDetectionResponse, error) {
	if req.GetDeployment() == nil {
		return nil, errox.InvalidArgs.CausedBy("deployment must be passed to deploy time detection")
	}
	if err := s.populateDeploymentWithClusterInfo(ctx, req.GetClusterId(), req.GetDeployment()); err != nil {
		return nil, err
	}

	// If we have enforcement only, then check if any of the policies need enforcement. If not, then just exit with no alerts generated
	if req.GetEnforcementOnly() {
		var evaluationRequired bool
		_ = s.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
			if isDeployTimeEnforcement(compiled.Policy().GetEnforcementActions()) {
				evaluationRequired = true
				return errors.New("not a real error, just early exits this foreach")
			}
			return nil
		})
		if !evaluationRequired {
			return &apiV1.DeployDetectionResponse{
				Runs: []*apiV1.DeployDetectionResponse_Run{
					{
						Name:   req.GetDeployment().GetName(),
						Type:   req.GetDeployment().GetType(),
						Alerts: nil,
					},
				},
			}, nil
		}
	}

	enrichmentCtx := enricher.EnrichmentContext{
		EnforcementOnly: req.GetEnforcementOnly(),
	}
	if req.GetNoExternalMetadata() {
		enrichmentCtx.FetchOpt = enricher.NoExternalMetadata
	}

	run, err := s.enrichAndDetect(ctx, enrichmentCtx, req.GetDeployment())
	if err != nil {
		return nil, err
	}
	return &apiV1.DeployDetectionResponse{
		Runs: []*apiV1.DeployDetectionResponse_Run{
			run,
		},
	}, nil
}

func getIgnoredObjectRefFromYAML(yaml string) (string, error) {
	unstructured, err := k8sutil.UnstructuredFromYAML(yaml)
	if err != nil {
		return "", err
	}
	return k8sobjects.RefOf(unstructured).String(), nil
}

// getFetchOptionFromRequest will return the associated enricher.FetchOption based on whether force or no external
// metadata is given.
// If both are specified, it will return an error since the combination is considered invalid (we cannot force a refetch
// and at the same time not take external metadata into account).
func getFetchOptionFromRequest(request interface {
	GetForce() bool
	GetNoExternalMetadata() bool
}) (enricher.FetchOption, error) {
	if request.GetForce() && request.GetNoExternalMetadata() {
		return enricher.UseCachesIfPossible, errox.InvalidArgs.New(
			"force option is incompatible with not fetching metadata from external sources")
	}
	if request.GetNoExternalMetadata() {
		return enricher.NoExternalMetadata, nil
	}
	if request.GetForce() {
		return enricher.UseImageNamesRefetchCachedValues, nil
	}
	return enricher.UseCachesIfPossible, nil
}
