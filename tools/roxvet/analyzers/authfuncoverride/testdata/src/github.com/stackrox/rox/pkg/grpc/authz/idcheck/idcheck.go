package idcheck

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type serviceAuth struct{}

func (serviceAuth) Authorized(context.Context, string) error { return nil }

func CentralOnly() authz.Authorizer           { return serviceAuth{} }
func SensorsOnly() authz.Authorizer           { return serviceAuth{} }
func SensorRegistrantsOnly() authz.Authorizer { return serviceAuth{} }
func ScannerOnly() authz.Authorizer           { return serviceAuth{} }
func ScannerV4IndexerOnly() authz.Authorizer  { return serviceAuth{} }
func ScannerV4MatcherOnly() authz.Authorizer  { return serviceAuth{} }
func CollectorOnly() authz.Authorizer         { return serviceAuth{} }
func AdmissionControlOnly() authz.Authorizer  { return serviceAuth{} }
