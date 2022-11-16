package store

import (
	routeV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/sensor/common/selector"
	v1 "k8s.io/api/core/v1"
)

// SelectorRouteWrap wraps a service with routes and selectors
type SelectorRouteWrap struct {
	*SelectorWrap
	Routes []*routeV1.Route
}

// SelectorWrap wraps a service with selectors
type SelectorWrap struct {
	*v1.Service
	Selector selector.Selector
}

// WrapService returns a service object with selector objects
func WrapService(svc *v1.Service) *SelectorWrap {
	return &SelectorWrap{
		Service:  svc,
		Selector: selector.CreateSelector(svc.Spec.Selector, selector.EmptyMatchesNothing()),
	}
}
