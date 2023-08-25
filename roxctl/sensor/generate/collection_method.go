package generate

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	humanReadableToEnum = map[string]storage.CollectionMethod{
		"default":  storage.CollectionMethod_UNSET_COLLECTION,
		"none":     storage.CollectionMethod_NO_COLLECTION,
		"ebpf":     storage.CollectionMethod_EBPF,
		"core_bpf": storage.CollectionMethod_CORE_BPF,
	}

	enumToHumanReadable = func() map[storage.CollectionMethod]string {
		m := make(map[storage.CollectionMethod]string)
		for k, v := range humanReadableToEnum {
			m[v] = k
		}
		return m
	}()
)

type collectionTypeWrapper struct {
	CollectionMethod *storage.CollectionMethod
}

func (f *collectionTypeWrapper) String() string {
	return enumToHumanReadable[*f.CollectionMethod]
}

func (f *collectionTypeWrapper) Set(input string) error {
	// For backwards compatibility.
	inputNormalized := strings.ToLower(input)
	switch inputNormalized {
	case "unset":
		// For backwards compatibility.
		inputNormalized = "default"
	}
	pt, ok := humanReadableToEnum[inputNormalized]
	if !ok {
		return errox.InvalidArgs.Newf("invalid collection method: %s", input)
	}
	*f.CollectionMethod = pt
	return nil
}

func (f *collectionTypeWrapper) Type() string {
	return "collection method"
}
