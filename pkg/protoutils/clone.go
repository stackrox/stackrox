package protoutils

import (
	"github.com/mauricelam/genny/generic"
)

// ProtoCloneType represents a generic proto type that we clone.
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -imp=github.com/stackrox/rox/generated/api/v1 -out=gen-$GOFILE gen "ProtoCloneType=*storage.Policy,*storage.Deployment,*storage.Alert,*v1.Query,*storage.Cluster"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/internalapi/central -out=gen-internalapi-$GOFILE gen "ProtoCloneType=*central.SensorEvent"
type ProtoCloneType generic.Type

// CloneProtoCloneType is a (generic) wrapper around proto.Clone that is strongly typed.
func CloneProtoCloneType(val ProtoCloneType) ProtoCloneType {
	return protoCloneWrapper(val).(ProtoCloneType)
}
