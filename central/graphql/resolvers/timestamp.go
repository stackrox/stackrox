package resolvers

import (
	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
)

func timestamp(ts *types.Timestamp) (*graphql.Time, error) {
	if ts == nil {
		return nil, nil
	}
	t, err := types.TimestampFromProto(ts)
	return &graphql.Time{Time: t}, err
}
