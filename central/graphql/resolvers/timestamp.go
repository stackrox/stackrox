package resolvers

import (
	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/protocompat"
)

func timestamp(ts *types.Timestamp) (*graphql.Time, error) {
	if ts == nil {
		return nil, nil
	}
	t, err := protocompat.ConvertTimestampToTimeOrError(ts)
	return &graphql.Time{Time: t}, err
}
