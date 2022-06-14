package upgradectx

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fallbackDecoder []runtime.Decoder

func (d fallbackDecoder) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	if len(d) == 0 {
		return nil, nil, errors.New("no decoders given")
	}

	errs := errorhelpers.NewErrorList("all decoding attempts failed")
	for _, dec := range d {
		obj, gvk, err := dec.Decode(data, defaults, into)
		if err == nil {
			return obj, gvk, nil
		}
		errs.AddError(err)
	}
	return nil, nil, errs.ToError()
}
