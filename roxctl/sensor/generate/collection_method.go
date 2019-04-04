package generate

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

var (
	humanReadableToEnum = map[string]storage.CollectionMethod{
		"unset":         storage.CollectionMethod_UNSET_COLLECTION,
		"none":          storage.CollectionMethod_NO_COLLECTION,
		"kernel-module": storage.CollectionMethod_KERNEL_MODULE,
		"ebpf":          storage.CollectionMethod_EBPF,
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
	pt, ok := humanReadableToEnum[strings.ToLower(input)]
	if !ok {
		return fmt.Errorf("Invalid collection method: %s", input)
	}
	*f.CollectionMethod = pt
	return nil
}

func (f *collectionTypeWrapper) Type() string {
	return "collection method"
}
