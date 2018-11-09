package protoutils

import (
	"github.com/mauricelam/genny/generic"
)

// ProtoCloneType represents a generic proto type that we clone.
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/v1 -out=gen-$GOFILE gen "ProtoCloneType=*v1.Policy,*v1.Deployment,*v1.Alert"
type ProtoCloneType generic.Type

// CloneProtoCloneType is a (generic) wrapper around proto.Clone that is strongly typed.
func CloneProtoCloneType(val ProtoCloneType) ProtoCloneType {
	return protoCloneWrapper(val).(ProtoCloneType)
}
