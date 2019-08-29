package preflight

import (
	"bytes"

	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

type schemaValidationCheck struct{}

func (schemaValidationCheck) Name() string {
	return "Schema validation"
}

func (schemaValidationCheck) Check(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	jsonEncoder := json.NewSerializer(json.DefaultMetaFactory, nil, nil, false)

	for _, act := range execPlan.Actions() {
		if act.Object == nil {
			continue
		}

		var buf bytes.Buffer
		if err := jsonEncoder.Encode(act.Object, &buf); err != nil {
			reporter.Errorf("Failed to serialize object %v to JSON: %v", act.ObjectRef, err)
			continue
		}

		if err := ctx.Validator().ValidateBytes(buf.Bytes()); err != nil {
			reporter.Errorf("Schema validation for object %v failed: %v", act.ObjectRef, err)
		}
	}

	return nil
}
