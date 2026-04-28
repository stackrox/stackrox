package views

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
)

var log = logging.LoggerForModule()

// ListAlertScanner holds pgtype scan destinations for the column projection
// query and converts scanned values into a *storage.ListAlert.
type ListAlertScanner struct {
	ID                 pgtype.Text
	LifecycleStage     pgtype.Int4
	ViolationTime      pgtype.Timestamp
	State              pgtype.Int4
	PolicyID           pgtype.Text
	PolicyName         pgtype.Text
	Severity           pgtype.Int4
	Description        pgtype.Text
	Categories         pgtype.FlatArray[string]
	EnforcementAction  pgtype.Int4
	EnforcementCount   pgtype.Int4
	EntityType         pgtype.Int4
	ClusterID          pgtype.Text
	ClusterName        pgtype.Text
	Namespace          pgtype.Text
	NamespaceID        pgtype.Text
	DeploymentID       pgtype.Text
	DeploymentName     pgtype.Text
	DeploymentType     pgtype.Text
	DeploymentInactive pgtype.Bool
	NodeID             pgtype.Text
	NodeName           pgtype.Text
	ResourceName       pgtype.Text
	ResourceType       pgtype.Int4
}

// Dests returns scan destination pointers in the order matching listAlertSelectProtos.
func (s *ListAlertScanner) Dests() []any {
	return []any{
		&s.ID, &s.LifecycleStage, &s.ViolationTime, &s.State,
		&s.PolicyID, &s.PolicyName, &s.Severity, &s.Description, &s.Categories,
		&s.EnforcementAction, &s.EnforcementCount, &s.EntityType,
		&s.ClusterID, &s.ClusterName, &s.Namespace, &s.NamespaceID,
		&s.DeploymentID, &s.DeploymentName, &s.DeploymentType, &s.DeploymentInactive,
		&s.NodeID, &s.NodeName, &s.ResourceName, &s.ResourceType,
	}
}

// Build converts the scanned column values into a *storage.ListAlert.
func (s *ListAlertScanner) Build() *storage.ListAlert {
	la := &storage.ListAlert{
		Id:             s.ID.String,
		LifecycleStage: storage.LifecycleStage(s.LifecycleStage.Int32),
		State:          storage.ViolationState(s.State.Int32),
		Policy: &storage.ListAlertPolicy{
			Id:          s.PolicyID.String,
			Name:        s.PolicyName.String,
			Severity:    storage.Severity(s.Severity.Int32),
			Description: s.Description.String,
			Categories:  []string(s.Categories),
		},
		EnforcementAction: storage.EnforcementAction(s.EnforcementAction.Int32),
	}

	if storage.ViolationState(s.State.Int32) == storage.ViolationState_ACTIVE {
		la.EnforcementCount = s.EnforcementCount.Int32
	}

	if s.ViolationTime.Valid {
		la.Time = protocompat.ConvertTimeToTimestampOrNil(&s.ViolationTime.Time)
	}

	et := storage.Alert_EntityType(s.EntityType.Int32)
	switch et {
	case storage.Alert_DEPLOYMENT:
		la.CommonEntityInfo = &storage.ListAlert_CommonEntityInfo{
			ClusterName:  s.ClusterName.String,
			ClusterId:    s.ClusterID.String,
			Namespace:    s.Namespace.String,
			NamespaceId:  s.NamespaceID.String,
			ResourceType: storage.ListAlert_DEPLOYMENT,
		}
		la.Entity = &storage.ListAlert_Deployment{
			Deployment: &storage.ListAlertDeployment{
				Id:             s.DeploymentID.String,
				Name:           s.DeploymentName.String,
				ClusterName:    s.ClusterName.String,
				ClusterId:      s.ClusterID.String,
				Namespace:      s.Namespace.String,
				NamespaceId:    s.NamespaceID.String,
				DeploymentType: s.DeploymentType.String,
				Inactive:       s.DeploymentInactive.Bool,
			},
		}
	case storage.Alert_RESOURCE:
		la.CommonEntityInfo = &storage.ListAlert_CommonEntityInfo{
			ClusterName:  s.ClusterName.String,
			ClusterId:    s.ClusterID.String,
			Namespace:    s.Namespace.String,
			NamespaceId:  s.NamespaceID.String,
			ResourceType: storage.ListAlert_ResourceType(s.ResourceType.Int32),
		}
		la.Entity = &storage.ListAlert_Resource{
			Resource: &storage.ListAlert_ResourceEntity{
				Name: s.ResourceName.String,
			},
		}
	case storage.Alert_NODE:
		la.CommonEntityInfo = &storage.ListAlert_CommonEntityInfo{
			ClusterName:  s.ClusterName.String,
			ClusterId:    s.ClusterID.String,
			ResourceType: storage.ListAlert_NODE,
		}
		la.Entity = &storage.ListAlert_Node{
			Node: &storage.ListAlert_NodeEntity{
				Name: s.NodeName.String,
			},
		}
	default:
		log.Warnf("alert %s has unhandled entity type %d, skipping entity info", s.ID.String, s.EntityType.Int32)
	}

	return la
}
