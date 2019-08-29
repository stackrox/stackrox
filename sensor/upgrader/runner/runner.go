package runner

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/bundle"
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/snapshot"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

type runner struct {
	ctx *upgradectx.UpgradeContext

	preUpgradeState      map[k8sobjects.ObjectRef]k8sobjects.Object
	postUpgradeWantState map[k8sobjects.ObjectRef]k8sobjects.Object
}

func (r *runner) Run() error {
	preUpgradeObjs, err := snapshot.TakeOrReadSnapshot(r.ctx)
	if err != nil {
		return err
	}
	r.preUpgradeState = k8sobjects.BuildObjectMap(preUpgradeObjs)

	bundleContents, err := bundle.FetchBundle(r.ctx)
	if err != nil {
		return err
	}

	postUpgradeObjs, err := bundle.InstantiateBundle(r.ctx, bundleContents)
	if err != nil {
		return err
	}
	transferMetadata(postUpgradeObjs, r.preUpgradeState)

	r.postUpgradeWantState = k8sobjects.BuildObjectMap(postUpgradeObjs)

	executionPlan, err := plan.GenerateExecutionPlan(r.ctx, postUpgradeObjs)
	if err != nil {
		return err
	}

	fmt.Println("Desired execution plan:")
	encoder := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
	fmt.Println("CREATIONS")
	for _, obj := range executionPlan.Creations {
		var strW strings.Builder
		utils.Must(encoder.Encode(obj, &strW))
		fmt.Println(strW.String())
		fmt.Println("---")
	}

	fmt.Println()
	fmt.Println("UPDATES")
	for _, obj := range executionPlan.Updates {
		var strW strings.Builder
		utils.Must(encoder.Encode(obj, &strW))
		fmt.Println(strW.String())
		fmt.Println("---")
	}

	fmt.Println()
	fmt.Println("DELETIONS")
	for _, objRef := range executionPlan.Deletions {
		fmt.Println(objRef)
	}

	return nil
}
