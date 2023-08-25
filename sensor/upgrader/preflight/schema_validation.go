package preflight

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubectl/pkg/validation"
)

var (
	defaultJSONEncoder = json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{})
)

type schemaValidationCheck struct{}

func (schemaValidationCheck) Name() string {
	return "Schema validation"
}

func (schemaValidationCheck) Check(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	for _, act := range execPlan.Actions() {
		if act.Object == nil {
			continue
		}

		if err := validateObject(act.Object, ctx.Validator()); err != nil {
			reporter.Errorf("Object %s: %s", act.ObjectRef, err)
		}
	}

	return nil
}

func validateObject(obj k8sutil.Object, validator validation.Schema) error {
	var buf bytes.Buffer
	if err := defaultJSONEncoder.Encode(obj, &buf); err != nil {
		return errors.Wrap(err, "failed to serialize to JSON")
	}

	if err := validator.ValidateBytes(buf.Bytes()); err != nil {
		return errors.Wrap(err, "schema validation failed")
	}
	return nil
}
