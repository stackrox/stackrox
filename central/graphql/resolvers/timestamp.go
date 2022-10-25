package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/transitional/protocompat/types"
)

func timestamp(ts *types.Timestamp) (*graphql.Time, error) {
	if ts == nil {
		return nil, nil
	}
	t, err := types.TimestampFromProto(ts)
	return &graphql.Time{Time: t}, err
}
