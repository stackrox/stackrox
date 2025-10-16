package observe

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"google.golang.org/protobuf/proto"
)

// CountAllowedTraces exists solely for testing purposes.
func CountAllowedTraces(trace *AuthzTrace) int {
	result := 0
	for _, v := range trace.allowed {
		result = result + int(v)
	}
	return result
}

func calculateRequest(ctx context.Context, rpcMethod string) *v1.AuthorizationTraceResponse_Request {
	var method, endpoint string
	if ri := requestinfo.FromContext(ctx); ri.HTTPRequest != nil {
		method = ri.HTTPRequest.Method
		endpoint = ri.HTTPRequest.URL.String()
	} else {
		method = "GRPC"
		endpoint = rpcMethod
	}

	request := &v1.AuthorizationTraceResponse_Request{}
	request.SetEndpoint(endpoint)
	request.SetMethod(method)

	return request
}

func calculateResponse(requestError error) *v1.AuthorizationTraceResponse_Response {
	response := &v1.AuthorizationTraceResponse_Response{}
	response.SetStatus(v1.AuthorizationTraceResponse_Response_SUCCESS)

	// requestError also includes AuthStatus.Error.
	if requestError != nil {
		response.SetStatus(v1.AuthorizationTraceResponse_Response_FAILURE)
		response.SetError(requestError.Error())
	}

	return response
}

func calculateUser(ctx context.Context) *v1.AuthorizationTraceResponse_User {
	id := authn.IdentityFromContextOrNil(ctx)
	if id == nil {
		return nil
	}

	roles := make([]*v1.AuthorizationTraceResponse_User_Role, 0, len(id.Roles()))
	for _, rr := range id.Roles() {
		r := &v1.AuthorizationTraceResponse_User_Role{}
		r.SetName(rr.GetRoleName())
		r.SetPermissions(rr.GetPermissions())
		r.SetAccessScopeName(rr.GetAccessScope().GetName())
		r.SetAccessScope(rr.GetAccessScope().GetRules())
		roles = append(roles, r)
	}

	user := &v1.AuthorizationTraceResponse_User{}
	user.SetUsername(id.User().GetUsername())
	user.SetFriendlyName(id.User().GetFriendlyName())
	user.SetAggregatedPermissions(id.Permissions())
	user.SetRoles(roles)
	return user
}

func calculateTrace(authzTrace *AuthzTrace) *v1.AuthorizationTraceResponse_Trace {
	if authzTrace == nil {
		return nil
	}

	authzTrace.mutex.Lock()
	defer authzTrace.mutex.Unlock()

	trace := &v1.AuthorizationTraceResponse_Trace{}
	trace.SetScopeCheckerType(string(authzTrace.sccType))
	if authzTrace.sccType == ScopeCheckerBuiltIn {
		atb := &v1.AuthorizationTraceResponse_Trace_BuiltInAuthorizer{}
		atb.SetClustersTotalNum(authzTrace.numClusters)
		atb.SetNamespacesTotalNum(authzTrace.numNamespaces)
		atb.SetDeniedAuthzDecisions(authzTrace.denied)
		atb.SetAllowedAuthzDecisions(authzTrace.allowed)
		atb.SetEffectiveAccessScopes(authzTrace.effectiveAccessScopes)
		trace.SetBuiltIn(proto.ValueOrDefault(atb))
	}

	return trace
}
