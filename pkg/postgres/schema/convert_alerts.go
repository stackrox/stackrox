package schema

import (
	"github.com/gogo/protobuf/proto"
	"github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

func ConvertAlertFromProto(p *storage.Alert) (*Alerts, error) {
	bytes, err := p.Marshal()
	if err != nil {
		return nil, err
	}
	model := &Alerts{
		Id:                       p.GetId(),
		PolicyId:                 p.GetPolicy().GetId(),
		PolicyName:               p.GetPolicy().GetName(),
		PolicyDescription:        p.GetPolicy().GetDescription(),
		PolicyDisabled:           p.GetPolicy().GetDisabled(),
		PolicyCategories:         pq.Array(p.GetPolicy().GetCategories()).(*pq.StringArray),
		PolicyLifecycleStages:    pq.Array(pgutils.ConvertEnumSliceToIntArray(p.GetPolicy().GetLifecycleStages())).(*pq.Int32Array),
		PolicySeverity:           p.GetPolicy().GetSeverity(),
		PolicyEnforcementActions: pq.Array(pgutils.ConvertEnumSliceToIntArray(p.GetPolicy().GetEnforcementActions())).(*pq.Int32Array),
		PolicyLastUpdated:        pgutils.NilOrTime(p.GetPolicy().GetLastUpdated()),
		PolicySORTName:           p.GetPolicy().GetSORTName(),
		PolicySORTLifecycleStage: p.GetPolicy().GetSORTLifecycleStage(),
		PolicySORTEnforcement:    p.GetPolicy().GetSORTEnforcement(),
		LifecycleStage:           p.GetLifecycleStage(),
		ClusterId:                p.GetClusterId(),
		ClusterName:              p.GetClusterName(),
		Namespace:                p.GetNamespace(),
		NamespaceId:              p.GetNamespaceId(),
		DeploymentId:             p.GetDeployment().GetId(),
		DeploymentName:           p.GetDeployment().GetName(),
		DeploymentInactive:       p.GetDeployment().GetInactive(),
		ImageId:                  p.GetImage().GetId(),
		ImageNameRegistry:        p.GetImage().GetName().GetRegistry(),
		ImageNameRemote:          p.GetImage().GetName().GetRemote(),
		ImageNameTag:             p.GetImage().GetName().GetTag(),
		ImageNameFullName:        p.GetImage().GetName().GetFullName(),
		ResourceResourceType:     p.GetResource().GetResourceType(),
		ResourceName:             p.GetResource().GetName(),
		EnforcementAction:        p.GetEnforcement().GetAction(),
		Time:                     pgutils.NilOrTime(p.GetTime()),
		State:                    p.GetState(),
		Serialized:               bytes,
	}
	return model, nil
}

func ConvertAlertToProto(m *Alerts) (*storage.Alert, error) {
	var msg storage.Alert
	if err := proto.Unmarshal(m.Serialized, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
