package generate

import (
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
)

type loadBalancerWrapper struct {
	LoadBalancerType *v1.LoadBalancerType
}

var lbStringToType = map[string]v1.LoadBalancerType{
	"none":  v1.LoadBalancerType_NONE,
	"np":    v1.LoadBalancerType_NODE_PORT,
	"lb":    v1.LoadBalancerType_LOAD_BALANCER,
	"route": v1.LoadBalancerType_ROUTE,
}

var lbEnumToString = func() map[v1.LoadBalancerType]string {
	m := make(map[v1.LoadBalancerType]string)
	for k, v := range lbStringToType {
		m[v] = k
	}
	return m
}()

func (f *loadBalancerWrapper) String() string {
	return lbEnumToString[*f.LoadBalancerType]
}

func (f *loadBalancerWrapper) Set(input string) error {
	if val, ok := lbStringToType[strings.ToLower(input)]; ok {
		*f.LoadBalancerType = val
		return nil
	}
	return errox.InvalidArgs.Newf("invalid load balancer type: %q", input)
}

func (f *loadBalancerWrapper) Type() string {
	return "load balancer type"
}
