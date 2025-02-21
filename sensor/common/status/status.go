package status

import "context"

import "github.com/stackrox/rox/sensor/common/centralid"

const clusterIDValue = "clusterID"

func DefaultContext() context.Context {
	ctx := context.WithValue(context.Background(), clusterIDValue, centralid.Get())
	return ctx
}
