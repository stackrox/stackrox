package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	deleConnection "github.com/stackrox/rox/central/delegatedregistryconfig/util/connection"
	"github.com/stackrox/rox/central/delegatedregistryconfig/util/imageintegration"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/endpoints"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/scanners"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/secrets"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		or.SensorOr(user.With(permissions.View(resources.Integration))): {
			"/v1.ImageIntegrationService/GetImageIntegration",
			"/v1.ImageIntegrationService/GetImageIntegrations",
		},
		user.With(permissions.Modify(resources.Integration)): {
			"/v1.ImageIntegrationService/PostImageIntegration",
			"/v1.ImageIntegrationService/PutImageIntegration",
			"/v1.ImageIntegrationService/TestImageIntegration",
			"/v1.ImageIntegrationService/DeleteImageIntegration",
			"/v1.ImageIntegrationService/UpdateImageIntegration",
			"/v1.ImageIntegrationService/TestUpdatedImageIntegration",
		},
	})
)

// ImageIntegrationService is the struct that manages the ImageIntegration API
type serviceImpl struct {
	v1.UnimplementedImageIntegrationServiceServer

	registryFactory    registries.Factory
	scannerFactory     scanners.Factory
	nodeEnricher       enricher.NodeEnricher
	integrationManager enrichment.Manager
	datastore          datastore.DataStore
	clusterDatastore   clusterDatastore.DataStore
	reprocessorLoop    reprocessor.Loop
	connManager        connection.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterImageIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterImageIntegrationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func scrubImageIntegration(i *storage.ImageIntegration) {
	secrets.ScrubSecretsFromStructWithReplacement(i, secrets.ScrubReplacementStr)
}

// GetImageIntegration returns the image integration given its ID.
func (s *serviceImpl) GetImageIntegration(ctx context.Context, request *v1.ResourceByID) (*storage.ImageIntegration, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "image integration id must be provided")
	}
	integration, exists, err := s.datastore.GetImageIntegration(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "image integration %s not found", request.GetId())
	}
	scrubImageIntegration(integration)
	return integration, nil
}

// GetImageIntegrations returns all image integrations.
func (s *serviceImpl) GetImageIntegrations(ctx context.Context, request *v1.GetImageIntegrationsRequest) (*v1.GetImageIntegrationsResponse, error) {
	integrations, err := s.datastore.GetImageIntegrations(ctx, request)
	if err != nil {
		return nil, err
	}

	identity := authn.IdentityFromContextOrNil(ctx)
	if identity != nil {
		svc := identity.Service()
		if svc != nil && svc.GetType() == storage.ServiceType_SENSOR_SERVICE {
			return &v1.GetImageIntegrationsResponse{Integrations: integrations}, nil
		}
	}

	// Remove secrets for other API accessors.
	for _, i := range integrations {
		scrubImageIntegration(i)
	}
	return &v1.GetImageIntegrationsResponse{Integrations: integrations}, nil
}

func sortCategories(categories []storage.ImageIntegrationCategory) {
	sort.SliceStable(categories, func(i, j int) bool {
		return int32(categories[i]) < int32(categories[j])
	})
}

func (s *serviceImpl) validateTestAndNormalize(ctx context.Context, request *storage.ImageIntegration) error {
	if err := s.validateIntegration(ctx, request); err != nil {
		return err
	}

	if !request.GetSkipTestIntegration() {
		if err := s.testImageIntegration(request); err != nil {
			return err
		}
	}

	sortCategories(request.Categories)
	return nil
}

// PutImageIntegration modifies a given image integration, without stored credential reconciliation
func (s *serviceImpl) PutImageIntegration(ctx context.Context, imageIntegration *storage.ImageIntegration) (*v1.Empty, error) {
	return s.UpdateImageIntegration(ctx, &v1.UpdateImageIntegrationRequest{Config: imageIntegration, UpdatePassword: true})
}

// PostImageIntegration creates an image integration.
func (s *serviceImpl) PostImageIntegration(ctx context.Context, request *storage.ImageIntegration) (*storage.ImageIntegration, error) {
	if request.GetId() != "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id field should be empty when posting a new image integration")
	}

	// Do not allow manual creation of a scanner v4 integration.
	if request.GetType() == scannerTypes.ScannerV4 {
		return nil, errors.Wrap(errox.InvalidArgs, "scanner V4 integrations cannot be manually created")
	}

	if err := s.validateTestAndNormalize(ctx, request); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	id, err := s.datastore.AddImageIntegration(ctx, request)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	request.Id = id

	if err := s.integrationManager.Upsert(request); err != nil {
		_ = s.datastore.RemoveImageIntegration(ctx, request.GetId())
		return nil, err
	}

	s.broadcastUpdate(ctx, request)
	s.reprocessorLoop.ShortCircuit()
	return request, nil
}

// DeleteImageIntegration removes a image integration given its ID.
func (s *serviceImpl) DeleteImageIntegration(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "image integration id must be provided")
	}

	// Pull the existing integration to determine if should broadcast the delete to sensors.
	// Do this to avoid sending blind delete messages to potentially many clusters unnecessarily.
	ii, existed, err := s.datastore.GetImageIntegration(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	// Do not allow manual deletion of a scanner v4 integration.
	if existed && ii.GetType() == scannerTypes.ScannerV4 {
		return nil, errors.Wrap(errox.InvalidArgs, "scanner V4 integrations cannot be deleted")
	}

	if err := s.datastore.RemoveImageIntegration(ctx, request.GetId()); err != nil {
		return nil, err
	}

	if err := s.integrationManager.Remove(request.GetId()); err != nil {
		return nil, err
	}

	if existed {
		s.broadcastDelete(ctx, ii)
	}

	return &v1.Empty{}, nil
}

// UpdateImageIntegration modifies a given image integration, with optional stored credential reconciliation.
func (s *serviceImpl) UpdateImageIntegration(ctx context.Context, request *v1.UpdateImageIntegrationRequest) (*v1.Empty, error) {
	if err := s.validateIntegration(ctx, request.GetConfig()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.reconcileUpdateImageIntegrationRequest(ctx, request); err != nil {
		return nil, err
	}
	if err := s.validateTestAndNormalize(ctx, request.GetConfig()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.datastore.UpdateImageIntegration(ctx, request.GetConfig()); err != nil {
		return nil, err
	}

	if err := s.integrationManager.Upsert(request.GetConfig()); err != nil {
		return nil, err
	}

	s.broadcastUpdate(ctx, request.GetConfig())
	s.reprocessorLoop.ShortCircuit()
	return &v1.Empty{}, nil
}

// TestImageIntegration checks if the given image integration is correctly configured, without using stored credential reconciliation.
func (s *serviceImpl) TestImageIntegration(ctx context.Context, imageIntegration *storage.ImageIntegration) (*v1.Empty, error) {
	return s.TestUpdatedImageIntegration(ctx, &v1.UpdateImageIntegrationRequest{Config: imageIntegration, UpdatePassword: true})
}

// TestUpdatedImageIntegration checks if the given image integration is correctly configured, with optional stored credential reconciliation.
func (s *serviceImpl) TestUpdatedImageIntegration(ctx context.Context, request *v1.UpdateImageIntegrationRequest) (*v1.Empty, error) {
	if err := s.validateIntegration(ctx, request.GetConfig()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.reconcileUpdateImageIntegrationRequest(ctx, request); err != nil {
		return nil, err
	}
	if err := s.testImageIntegration(request.GetConfig()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) testImageIntegration(request *storage.ImageIntegration) error {
	for _, category := range request.GetCategories() {
		if category == storage.ImageIntegrationCategory_REGISTRY {
			if err := s.testRegistryIntegration(request); err != nil {
				return errors.Wrap(errox.InvalidArgs, errors.Wrap(err, "registry integration").Error())
			}
		}
		if category == storage.ImageIntegrationCategory_SCANNER {
			if err := s.testScannerIntegration(request); err != nil {
				return errors.Wrap(errox.InvalidArgs, errors.Wrap(err, "image scanner integration").Error())
			}
		}
		if category == storage.ImageIntegrationCategory_NODE_SCANNER {
			nodeIntegration, err := imageIntegrationToNodeIntegration(request)
			if err != nil {
				return errors.Wrap(errox.InvalidArgs, errors.Wrap(err, "node scanner integration").Error())
			}
			if err := s.testNodeScannerIntegration(nodeIntegration); err != nil {
				return errors.Wrap(errox.InvalidArgs, errors.Wrap(err, "node scanner integration").Error())
			}
		}
	}
	return nil
}

func (s *serviceImpl) testRegistryIntegration(integration *storage.ImageIntegration) error {
	registry, err := s.registryFactory.CreateRegistry(integration)
	if err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := registry.Test(); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return nil
}

func (s *serviceImpl) testScannerIntegration(integration *storage.ImageIntegration) error {
	scanner, err := s.scannerFactory.CreateScanner(integration)
	if err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := scanner.GetScanner().Test(); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return nil
}

func (s *serviceImpl) testNodeScannerIntegration(integration *storage.NodeIntegration) error {
	scanner, err := s.nodeEnricher.CreateNodeScanner(integration)
	if err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := scanner.GetNodeScanner().TestNodeScanner(); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return nil
}

func (s *serviceImpl) validateIntegration(ctx context.Context, request *storage.ImageIntegration) error {
	if request == nil {
		return errors.New("empty integration")
	}
	errorList := errorhelpers.NewErrorList("Validation")
	if err := endpoints.ValidateEndpoints(request.IntegrationConfig); err != nil {
		errorList.AddWrap(err, "invalid endpoint")
	}
	if len(request.GetCategories()) == 0 {
		errorList.AddStrings("integrations require a category")
	}

	// Validate if there is a name. If there isn't, then skip the DB name check by returning the accumulated errors
	if request.GetName() == "" {
		errorList.AddString("name for integration is required")
		return errorList.ToError()
	}

	integrations, err := s.datastore.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{Name: request.GetName()})
	if err != nil {
		return err
	}
	if len(integrations) != 0 && request.GetId() != integrations[0].GetId() {
		errorList.AddStringf("integration with name %q already exists", request.GetName())
	}
	return errorList.ToError()
}

func (s *serviceImpl) reconcileUpdateImageIntegrationRequest(ctx context.Context, updateRequest *v1.UpdateImageIntegrationRequest) error {
	if updateRequest.GetUpdatePassword() {
		return nil
	}
	if updateRequest.GetConfig() == nil {
		return errors.Wrap(errox.InvalidArgs, "request is missing image integration config")
	}
	if updateRequest.GetConfig().GetId() == "" {
		return errors.Wrap(errox.InvalidArgs, "id required for stored credential reconciliation")
	}
	integration, exists, err := s.datastore.GetImageIntegration(ctx, updateRequest.GetConfig().GetId())
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "image integration %s not found", updateRequest.GetConfig().GetId())
	}
	if err := s.reconcileImageIntegrationWithExisting(updateRequest.GetConfig(), integration); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return nil
}

func (s *serviceImpl) reconcileImageIntegrationWithExisting(updatedConfig, storedConfig *storage.ImageIntegration) error {
	if updatedConfig.GetIntegrationConfig() == nil {
		return errors.New("the request doesn't have a valid integration config type")
	}
	return secrets.ReconcileScrubbedStructWithExisting(updatedConfig, storedConfig)
}

// broadcastUpdate will send the updated image integration to all Sensors that can handle
// the update if the integration is valid for broadcasting (not auto generated, is a registry, etc.).
func (s *serviceImpl) broadcastUpdate(ctx context.Context, ii *storage.ImageIntegration) {
	if !imageintegration.ValidForSync(ii) {
		return
	}

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ImageIntegrations{
			ImageIntegrations: &central.ImageIntegrations{
				UpdatedIntegrations: []*storage.ImageIntegration{ii},
			},
		},
	}

	s.broadcast(ctx, "updated", ii, msg)
}

// broadcastDelete will send the deleted image integration to all Sensors that can handle
// the update.
func (s *serviceImpl) broadcastDelete(ctx context.Context, ii *storage.ImageIntegration) {
	if !imageintegration.ValidForSync(ii) {
		return
	}

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ImageIntegrations{
			ImageIntegrations: &central.ImageIntegrations{
				DeletedIntegrationIds: []string{ii.GetId()},
			},
		},
	}

	s.broadcast(ctx, "deleted", ii, msg)
}

func (s *serviceImpl) broadcast(ctx context.Context, action string, ii *storage.ImageIntegration, msg *central.MsgToSensor) {
	log.Infof("Broadcasting %v image integration %q (%v)", action, ii.GetName(), ii.GetId())
	for _, conn := range s.connManager.GetActiveConnections() {
		if !deleConnection.ValidForDelegation(conn) {
			continue
		}

		clusterID := conn.ClusterID()

		log.Debugf("Sending %v image integration %q (%v) to cluster %q", action, ii.GetName(), ii.GetId(), clusterID)

		err := conn.InjectMessage(ctx, msg)
		if err != nil {
			log.Warnf("Failed to send %v image integration %q (%v) to cluster %q", action, ii.GetName(), ii.GetId(), clusterID)
		}
	}
}
