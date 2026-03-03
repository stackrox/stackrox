package audit

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	tokenServiceV1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	auditPkg "github.com/stackrox/rox/pkg/audit"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/interceptor"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
)

const (
	defaultGRPCMethod = "GRPC"

	refererKey = "Referer"
)

// audit handles the creation of auditPkg logs from gRPC requests that aren't GETs
// currently, it only handles grpc because we do not do anything substantial on HTTP Post
type audit struct {
	notifications      notifier.Processor
	withoutPermissions bool
}

// New takes in a processor and returns an audit struct
func New(notifications notifier.Processor) auditPkg.Auditor {
	return &audit{
		notifications:      notifications,
		withoutPermissions: env.AuditLogWithoutPermissions.BooleanSetting(),
	}
}

// SendAuditMessage will send an audit message for the specified request.
func (a *audit) SendAuditMessage(ctx context.Context, req interface{}, grpcMethod string,
	authError interceptor.AuthStatus, requestError error) {
	if !a.notifications.HasEnabledAuditNotifiers() {
		return
	}

	am := a.newAuditMessage(ctx, req, grpcMethod, authError, requestError)
	if am == nil {
		return
	}
	a.notifications.ProcessAuditMessage(ctx, am)
}

var (
	requestInteractionMap = map[string]v1.Audit_Interaction{
		http.MethodPost:   v1.Audit_CREATE,
		http.MethodPut:    v1.Audit_UPDATE,
		http.MethodPatch:  v1.Audit_UPDATE,
		http.MethodDelete: v1.Audit_DELETE,
		defaultGRPCMethod: v1.Audit_UPDATE,
	}

	// auditableServiceEndpoints contains service-to-service endpoints that should be audited.
	// When adding new security-sensitive internal RPCs, add their full method name constant
	// here to ensure they appear in audit logs. These use generated gRPC full method name
	// constants, so any rename or signature change to the RPC will cause a compile-time
	// error rather than silently breaking auditing.
	auditableServiceEndpoints = set.NewFrozenSet(
		tokenServiceV1.TokenService_GenerateTokenForPermissionsAndScope_FullMethodName,
	)
)

func (a *audit) newAuditMessage(ctx context.Context, req interface{}, grpcFullMethod string,
	authError interceptor.AuthStatus, requestError error) *v1.Audit_Message {
	ri := requestinfo.FromContext(ctx)

	msg := &v1.Audit_Message{
		Time: protocompat.TimestampNow(),
	}

	identity := authn.IdentityFromContextOrNil(ctx)
	isServiceIdentity := false
	// Selectively audit service-to-service requests for security-sensitive operations.
	if identity != nil {
		if svc := identity.Service(); svc != nil {
			isServiceIdentity = true
			if !auditableServiceEndpoints.Contains(grpcFullMethod) {
				return nil
			}
			// For service requests, populate user info from service identity.
			msg.User = serviceIdentityUserInfo(svc)
		} else {
			msg.User = utils.IfThenElse(a.withoutPermissions,
				stripPermissionsFromUserInfo(identity.User()),
				identity.User(),
			)
		}
	}

	var method, endpoint string
	if ri.HTTPRequest != nil {
		method = ri.HTTPRequest.Method
		endpoint = ri.HTTPRequest.URL.String()
		if referer := ri.HTTPRequest.Headers.Get(refererKey); referer != "" {
			msg.Method = v1.Audit_UI
		} else {
			msg.Method = v1.Audit_API
		}
	} else {
		method = defaultGRPCMethod
		endpoint = grpcFullMethod
		// Service-to-service gRPC calls should be marked as API, not CLI.
		if isServiceIdentity {
			msg.Method = v1.Audit_API
		} else {
			msg.Method = v1.Audit_CLI
		}
	}

	interaction, ok := requestInteractionMap[method]
	if !ok {
		return nil
	}
	msg.Interaction = interaction

	msg.Request = &v1.Audit_Message_Request{
		Endpoint: endpoint,
		Method:   method,
		Payload:  protoutils.RequestToAny(req),
		SourceHeaders: &v1.Audit_Message_Request_SourceHeaders{
			XForwardedFor: ri.Source.XForwardedFor,
			RemoteAddr:    ri.Source.RemoteAddr,
			RequestAddr:   ri.Source.RequestAddr,
		},
		SourceIp: ri.Source.GetSourceIP(),
	}

	msg.Status, msg.StatusReason = calculateAuditStatus(authError, requestError)
	return msg
}

// UnaryServerInterceptor is the interceptor for audit logging
func (a *audit) UnaryServerInterceptor() func(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		go a.SendAuditMessage(ctx, req, info.FullMethod, interceptor.GetAuthErrorFromContext(ctx), err)
		return resp, err
	}
}

// PostAuthHTTPInterceptor is the interceptor for audit logging after the route authorization handler.
func (a *audit) PostAuthHTTPInterceptor(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statusTrackingWriter := httputil.NewStatusTrackingWriter(w)
		handler.ServeHTTP(statusTrackingWriter, r)

		go a.SendAuditMessage(r.Context(), r, r.RequestURI, interceptor.AuthStatus{},
			statusTrackingWriter.GetStatusCodeError())
	})
}

func calculateAuditStatus(authError interceptor.AuthStatus, requestError error) (v1.Audit_RequestStatus, string) {
	switch {
	case authError.Error != nil:
		return v1.Audit_AUTH_FAILED, authError.String()
	case requestError != nil && errors.Is(requestError, sac.ErrResourceAccessDenied):
		return v1.Audit_AUTH_FAILED, requestError.Error()
	case requestError != nil:
		return v1.Audit_REQUEST_FAILED, requestError.Error()
	default:
		return v1.Audit_REQUEST_SUCCEEDED, ""
	}
}

// serviceIdentityUserInfo builds a UserInfo from a service identity for audit logging.
// Username uses the "service:<type>:<id>" format to clearly distinguish
// service identities from regular users in audit log queries.
func serviceIdentityUserInfo(svc *storage.ServiceIdentity) *storage.UserInfo {
	return &storage.UserInfo{
		Username:     fmt.Sprintf("service:%s:%s", svc.GetType().String(), svc.GetId()),
		FriendlyName: fmt.Sprintf("Service: %s (ID: %s)", svc.GetType().String(), svc.GetId()),
	}
}

func stripPermissionsFromUserInfo(userInfo *storage.UserInfo) *storage.UserInfo {
	userInfoWithoutPermissions := userInfo.CloneVT()
	userInfoWithoutPermissions.Permissions = nil

	userRolesWithoutPermissions := make([]*storage.UserInfo_Role, 0, len(userInfo.GetRoles()))
	for _, userRole := range userInfo.GetRoles() {
		userRolesWithoutPermissions = append(userRolesWithoutPermissions, &storage.UserInfo_Role{
			Name: userRole.GetName(),
		})
	}
	userInfoWithoutPermissions.Roles = userRolesWithoutPermissions

	return userInfoWithoutPermissions
}
