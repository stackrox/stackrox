package c

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	authzand "github.com/stackrox/rox/pkg/grpc/authz/and"
	"github.com/stackrox/rox/pkg/grpc/authz/deny"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	authzor "github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

// Package-level vars for allowed constructors.
var authScannerV4Or = authzor.ScannerV4()
var authSensorOr = authzor.SensorOr(user.WithRole("Admin"))
var authScannerOr = authzor.ScannerOr(idcheck.CentralOnly())
var authOrComposite = authzor.Or(user.WithRole("Admin"), allow.Anonymous())
var authAndComposite = authzand.And(user.WithRole("Admin"), idcheck.CentralOnly())
var authWithRole = user.WithRole("Admin")
var authWith = user.With()
var authAuthenticated = user.Authenticated()
var authAnon = allow.Anonymous()
var authDeny = deny.Everyone()
var authCentral = idcheck.CentralOnly()
var authSensors = idcheck.SensorsOnly()
var authSensorReg = idcheck.SensorRegistrantsOnly()
var authScanner = idcheck.ScannerOnly()
var authScannerV4Indexer = idcheck.ScannerV4IndexerOnly()
var authScannerV4Matcher = idcheck.ScannerV4MatcherOnly()
var authCollector = idcheck.CollectorOnly()
var authAdmCtrl = idcheck.AdmissionControlOnly()
var authPerRPC = perrpc.FromMap(map[authz.Authorizer][]string{
	user.WithRole("Admin"): {"/svc/MethodA"},
	allow.Anonymous():      {"/svc/MethodB"},
	deny.Everyone():        {"/svc/MethodC"},
	idcheck.SensorsOnly():  {"/svc/MethodD"},
})

// Named variable used as a perrpc.FromMap key.
var namedKey = user.WithRole("ReadOnly")
var authPerRPCNamed = perrpc.FromMap(map[authz.Authorizer][]string{
	namedKey: {"/svc/MethodE"},
})

// Custom type that does NOT come from an allowed constructor.
type customAuth struct{}

func (customAuth) Authorized(_ context.Context, _ string) error { return nil }

var badAuth = customAuth{}

// ---- Correct: user.WithRole ----

type svc1 struct{}

func (s *svc1) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authWithRole.Authorized(ctx, fullMethodName)
}

// ---- Correct: user.With ----

type svc2 struct{}

func (s *svc2) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authWith.Authorized(ctx, fullMethodName)
}

// ---- Correct: user.Authenticated ----

type svc3 struct{}

func (s *svc3) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authAuthenticated.Authorized(ctx, fullMethodName)
}

// ---- Correct: allow.Anonymous ----

type svc4 struct{}

func (s *svc4) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authAnon.Authorized(ctx, fullMethodName)
}

// ---- Correct: deny.Everyone ----

type svc5 struct{}

func (s *svc5) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authDeny.Authorized(ctx, fullMethodName)
}

// ---- Correct: idcheck constructors (one per kind) ----

type svc6 struct{}

func (s *svc6) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authCentral.Authorized(ctx, fullMethodName)
}

type svc7 struct{}

func (s *svc7) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authSensors.Authorized(ctx, fullMethodName)
}

type svc8 struct{}

func (s *svc8) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authSensorReg.Authorized(ctx, fullMethodName)
}

type svc9 struct{}

func (s *svc9) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authScanner.Authorized(ctx, fullMethodName)
}

type svc10 struct{}

func (s *svc10) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authScannerV4Indexer.Authorized(ctx, fullMethodName)
}

type svc11 struct{}

func (s *svc11) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authScannerV4Matcher.Authorized(ctx, fullMethodName)
}

type svc12 struct{}

func (s *svc12) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authCollector.Authorized(ctx, fullMethodName)
}

type svc13 struct{}

func (s *svc13) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authAdmCtrl.Authorized(ctx, fullMethodName)
}

// ---- Fail: inline idcheck calls (not a package-level variable) ----

type svc14 struct{}

func (s *svc14) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

type svc15 struct{}

func (s *svc15) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, idcheck.AdmissionControlOnly().Authorized(ctx, fullMethodName)
}

// ---- Fail: inline deny.Everyone (not a package-level variable) ----

type svc16 struct{}

func (s *svc16) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, deny.Everyone().Authorized(ctx, fullMethodName)
}

// ---- Correct: perrpc.FromMap with all-valid inline keys ----

type svc17 struct{}

func (s *svc17) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authPerRPC.Authorized(ctx, fullMethodName)
}

// ---- Fail: inline perrpc.FromMap (not a package-level variable) ----

type svc18 struct{}

func (s *svc18) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, perrpc.FromMap(map[authz.Authorizer][]string{
		user.WithRole("Admin"):         {"/svc/MethodA"},
		idcheck.ScannerV4IndexerOnly(): {"/svc/MethodB"},
		deny.Everyone():                {"/svc/MethodC"},
	}).Authorized(ctx, fullMethodName)
}

// ---- Correct: perrpc.FromMap with a named-variable key ----

type svc19 struct{}

func (s *svc19) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authPerRPCNamed.Authorized(ctx, fullMethodName)
}

// ---- Correct: or.ScannerV4 via named var ----

type svc20 struct{}

func (s *svc20) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authScannerV4Or.Authorized(ctx, fullMethodName)
}

// ---- Fail: inline or.ScannerV4 (not a package-level variable) ----

type svc21 struct{}

func (s *svc21) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, authzor.ScannerV4().Authorized(ctx, fullMethodName)
}

// ---- Correct: or.SensorOr via named var ----

type svcSensorOrNamed struct{}

func (s *svcSensorOrNamed) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authSensorOr.Authorized(ctx, fullMethodName)
}

// ---- Fail: inline or.SensorOr (not a package-level variable) ----

type svcSensorOrInline struct{}

func (s *svcSensorOrInline) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, authzor.SensorOr(user.WithRole("ReadOnly")).Authorized(ctx, fullMethodName)
}

// ---- Correct: or.ScannerOr via named var ----

type svcScannerOrNamed struct{}

func (s *svcScannerOrNamed) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authScannerOr.Authorized(ctx, fullMethodName)
}

// ---- Fail: inline or.ScannerOr (not a package-level variable) ----

type svcScannerOrInline struct{}

func (s *svcScannerOrInline) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, authzor.ScannerOr(allow.Anonymous()).Authorized(ctx, fullMethodName)
}

// ---- Fail: custom authorizer ----

type svc22 struct{}

func (s *svc22) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, badAuth.Authorized(ctx, fullMethodName)
}

// ---- Correct: or.Or via named var ----

type svcOrNamed struct{}

func (s *svcOrNamed) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authOrComposite.Authorized(ctx, fullMethodName)
}

// ---- Fail: inline or.Or (not a package-level variable) ----

type svcOrInline struct{}

func (s *svcOrInline) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, authzor.Or(user.WithRole("Admin"), deny.Everyone(), idcheck.SensorsOnly()).Authorized(ctx, fullMethodName)
}

// ---- Correct: and.And via named var ----

type svcAndNamed struct{}

func (s *svcAndNamed) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authAndComposite.Authorized(ctx, fullMethodName)
}

// ---- Fail: inline and.And (not a package-level variable) ----

type svcAndInline struct{}

func (s *svcAndInline) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, authzand.And(user.Authenticated(), allow.Anonymous()).Authorized(ctx, fullMethodName)
}

// ---- Fail: or.Or with one invalid arg ----

type svcOrBad struct{}

func (s *svcOrBad) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, authzor.Or(user.WithRole("Admin"), badAuth).Authorized(ctx, fullMethodName)
}

// ---- Fail: and.And with one invalid arg ----

type svcAndBad struct{}

func (s *svcAndBad) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, authzand.And(badAuth, idcheck.CentralOnly()).Authorized(ctx, fullMethodName)
}

// ---- Fail: or.SensorOr with invalid inner authorizer ----

type svcSensorOrBad struct{}

func (s *svcSensorOrBad) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, authzor.SensorOr(badAuth).Authorized(ctx, fullMethodName)
}

// ---- Fail: perrpc.FromMap with one invalid inline key ----

type svc23 struct{}

func (s *svc23) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, perrpc.FromMap(map[authz.Authorizer][]string{
		user.WithRole("Admin"): {"/svc/MethodA"},
		badAuth:                {"/svc/MethodB"},
	}).Authorized(ctx, fullMethodName)
}

// ---- Fail: perrpc.FromMap where a named-variable key is an invalid authorizer ----

var badNamedKey = customAuth{}

type svc24 struct{}

func (s *svc24) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, perrpc.FromMap(map[authz.Authorizer][]string{
		badNamedKey: {"/svc/MethodA"},
	}).Authorized(ctx, fullMethodName)
}

// ---- Fail: package-level variable that is reassigned in an init function ----

var reassignableAuth authz.Authorizer = user.WithRole("Admin")

func init() {
	reassignableAuth = user.WithRole("User")
}

type svc25 struct{}

func (s *svc25) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, reassignableAuth.Authorized(ctx, fullMethodName)
}
