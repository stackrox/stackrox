package walker

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

func TestWalker(t *testing.T) {
	schema := Walk(reflect.TypeOf((*storage.Deployment)(nil)), "deployments")

	fmt.Println(schema)

	//table := Walk(reflect.TypeOf((*storage.Deployment)(nil)))

	//table.Print("", true)
	//searchPrint(table)
	//createTables(table)
	//insertObject(table, nil, 0)
	//generateInsertFunctions(table)
}

/*
	_, err = tx.Exec(context.Background(), "delete from container_normalized where parent_deployment_id = $1 and container_idx >= $2", dep.GetId(), len(dep.GetContainers()))
*/
