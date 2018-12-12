package protoutils

import (
	"github.com/mauricelam/genny/generic"
)

// ProtoCloneType represents a generic proto type that we clone.
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/api/v1 -imp=github.com/stackrox/rox/generated/storage -out=gen-$GOFILE gen "ProtoCloneType=*storage.Policy,*storage.Deployment,*v1.Alert"
type ProtoCloneType generic.Type

// CloneProtoCloneType is a (generic) wrapper around proto.Clone that is strongly typed.
func CloneProtoCloneType(val ProtoCloneType) ProtoCloneType {
	return protoCloneWrapper(val).(ProtoCloneType)
}
