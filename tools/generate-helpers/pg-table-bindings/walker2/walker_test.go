package walker

import (
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

func TestWalker(t *testing.T) {
	schema := Walk(reflect.TypeOf((*storage.Deployment)(nil)), "deployments")
	schema.Print()
}

/*
	_, err = tx.Exec(context.Background(), "delete from container_normalized where parent_deployment_id = $1 and container_idx >= $2", dep.GetId(), len(dep.GetContainers()))
*/
