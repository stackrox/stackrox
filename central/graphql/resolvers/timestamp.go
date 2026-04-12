package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// timestamp converts a proto timestamp to a graphql Time, or returns an error.
// This lives here (not in pkg/protocompat) to avoid pulling the graphql-go
// library into non-Central binaries like sensor and admission-control.
func timestamp(pbTime *timestamppb.Timestamp) (*graphql.Time, error) {
	if pbTime == nil {
		return nil, nil
	}
	return &graphql.Time{Time: pbTime.AsTime()}, pbTime.CheckValid()
}
