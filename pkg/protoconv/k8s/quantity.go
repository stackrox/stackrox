package k8s

import (
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	megabyte = 1024 * 1024
)

// ConvertQuantityToCores converts a quantity representing a number of cores
// into a float32.
func ConvertQuantityToCores(q *resource.Quantity) float32 {
	// kubernetes does not like floating point values so they make you jump through hoops
	f, err := strconv.ParseFloat(q.AsDec().String(), 32)
	if err != nil {
		log.Error(err)
	}
	return float32(f)
}

// ConvertCoresToQuantity converts a number of cores to a K8s quantity.
// Note that it only supports 3 digits of precision after the decimal point;
// any further digits will be lost.
func ConvertCoresToQuantity(cores float32) *resource.Quantity {
	return resource.NewMilliQuantity(int64(cores*1000), resource.DecimalSI)
}

// ConvertQuantityToMB converts a Kubernetes quantity representing an amount of memory,
// in bytes, to a number of MB.
func ConvertQuantityToMB(q *resource.Quantity) float32 {
	return float32(float64(q.Value()) / megabyte)
}

// ConvertMBToQuantity converts an amount of MB to a Kubernetes quantity
// representing an amount of memory in bytes. The returned quantity can
// be used to set memory requests/limits for a K8s object.
func ConvertMBToQuantity(mb float32) *resource.Quantity {
	return resource.NewQuantity(int64(megabyte*mb), resource.BinarySI)
}
