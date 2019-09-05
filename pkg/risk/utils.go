package risk

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	log = logging.LoggerForModule()
)

// BuildRiskProtoForEntity return risk proto will entity meta for given protobuf-generated objects
func BuildRiskProtoForEntity(msg proto.Message) *storage.Risk {
	risk := &storage.Risk{}
	switch msg := msg.(type) {
	case *storage.Deployment:
		return BuildRiskProtoForDeployment(msg)
	case *storage.Image:
		return BuildRiskProtoForImage(msg)
	case *storage.ServiceAccount:
		return BuildRiskProtoForServiceAccount(msg)
	default:
		log.Errorf("cannot build risk proto for %T type", msg)
	}
	return risk
}

// BuildRiskProtoForDeployment return risk proto will entity meta for given deployment
func BuildRiskProtoForDeployment(deployment *storage.Deployment) *storage.Risk {
	risk := &storage.Risk{
		Entity: &storage.RiskEntityMeta{
			Id:          deployment.GetId(),
			Namespace:   deployment.GetNamespace(),
			NamespaceId: deployment.GetNamespaceId(),
			ClusterId:   deployment.GetClusterId(),
			Type:        storage.RiskEntityType_DEPLOYMENT,
		},
	}
	risk.Id = getIDFromEntity(risk.GetEntity())
	return risk
}

// BuildRiskProtoForImage return risk proto will entity meta for given image
func BuildRiskProtoForImage(image *storage.Image) *storage.Risk {
	risk := &storage.Risk{
		Entity: &storage.RiskEntityMeta{
			Id:   image.GetId(),
			Type: storage.RiskEntityType_IMAGE,
		},
	}
	risk.Id = getIDFromEntity(risk.GetEntity())
	return risk
}

// BuildRiskProtoForServiceAccount return risk proto will entity meta for given service account
func BuildRiskProtoForServiceAccount(serviceAcc *storage.ServiceAccount) *storage.Risk {
	risk := &storage.Risk{
		Entity: &storage.RiskEntityMeta{
			Id:        serviceAcc.GetId(),
			Namespace: serviceAcc.GetNamespace(),
			ClusterId: serviceAcc.GetClusterId(),
			Type:      storage.RiskEntityType_SERVICEACCOUNT,
		},
	}
	risk.Id = getIDFromEntity(risk.GetEntity())
	return risk
}

func getIDFromEntity(entity *storage.RiskEntityMeta) string {
	id, err := GetID(entity.GetId(), entity.GetType())
	if err != nil {
		log.Error(err)
		return ""
	}
	return id
}

// GetID generates risk ID from risk ubject ID (e.g. deployment ID) and type.
func GetID(entityID string, entityType storage.RiskEntityType) (string, error) {
	if stringutils.AllNotEmpty(entityID, entityType.String()) {
		return fmt.Sprintf("%s:%s", strings.ToLower(entityType.String()), entityID), nil
	}
	return "", errors.New("cannot build risk ID")
}

// GetIDParts returns entity type and entity ID from risk ID.
func GetIDParts(riskID string) (storage.RiskEntityType, string, error) {
	idParts := strings.SplitN(riskID, ":", 2)
	if len(idParts) != 2 {
		return storage.RiskEntityType_UNKNOWN, "", errors.New("cannot extract id parts")
	}
	entityType, err := EntityType(idParts[0])
	return entityType, idParts[1], err
}

// EntityType returns enum of supplied entity type string.
func EntityType(entityType string) (storage.RiskEntityType, error) {
	value, found := storage.RiskEntityType_value[strings.ToUpper(entityType)]
	if !found {
		return storage.RiskEntityType_UNKNOWN, errors.New("unknown entity type")
	}

	return storage.RiskEntityType(value), nil
}
