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
