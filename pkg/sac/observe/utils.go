package observe

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc/authn"
	"github.com/stackrox/stackrox/pkg/grpc/requestinfo"
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

	request := &v1.AuthorizationTraceResponse_Request{
		Endpoint: endpoint,
		Method:   method,
	}

	return request
}

func calculateResponse(requestError error) *v1.AuthorizationTraceResponse_Response {
	response := &v1.AuthorizationTraceResponse_Response{
		Status: v1.AuthorizationTraceResponse_Response_SUCCESS,
	}

	// requestError also includes AuthStatus.Error.
	if requestError != nil {
		response.Status = v1.AuthorizationTraceResponse_Response_FAILURE
		response.Error = requestError.Error()
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
		r := &v1.AuthorizationTraceResponse_User_Role{
			Name:            rr.GetRoleName(),
			Permissions:     rr.GetPermissions(),
			AccessScopeName: rr.GetAccessScope().GetName(),
			AccessScope:     rr.GetAccessScope().GetRules(),
		}
		roles = append(roles, r)
	}

	user := &v1.AuthorizationTraceResponse_User{
		Username:              id.User().GetUsername(),
		FriendlyName:          id.User().GetFriendlyName(),
		AggregatedPermissions: id.Permissions(),
		Roles:                 roles,
	}
	return user
}

func calculateTrace(authzTrace *AuthzTrace) *v1.AuthorizationTraceResponse_Trace {
	if authzTrace == nil {
		return nil
	}

	authzTrace.mutex.Lock()
	defer authzTrace.mutex.Unlock()

	trace := &v1.AuthorizationTraceResponse_Trace{
		ScopeCheckerType: string(authzTrace.sccType),
	}
	if authzTrace.sccType == ScopeCheckerBuiltIn {
		trace.Authorizer = &v1.AuthorizationTraceResponse_Trace_BuiltIn{
			BuiltIn: &v1.AuthorizationTraceResponse_Trace_BuiltInAuthorizer{
				ClustersTotalNum:      authzTrace.numClusters,
				NamespacesTotalNum:    authzTrace.numNamespaces,
				DeniedAuthzDecisions:  authzTrace.denied,
				AllowedAuthzDecisions: authzTrace.allowed,
				EffectiveAccessScopes: authzTrace.effectiveAccessScopes,
			},
		}
	}

	return trace
}
