package audit

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

var requestInteractionMap = map[string]v1.Audit_Interaction{
	http.MethodPost:   v1.Audit_CREATE,
	http.MethodPut:    v1.Audit_UPDATE,
	http.MethodPatch:  v1.Audit_UPDATE,
	http.MethodDelete: v1.Audit_DELETE,
	defaultGRPCMethod: v1.Audit_UPDATE,
}

func (a *audit) newAuditMessage(ctx context.Context, req interface{}, grpcFullMethod string,
	authError interceptor.AuthStatus, requestError error) *v1.Audit_Message {
	ri := requestinfo.FromContext(ctx)

	msg := &v1.Audit_Message{}
	msg.SetTime(protocompat.TimestampNow())

	identity := authn.IdentityFromContextOrNil(ctx)
	// Ignore requests from services
	if identity != nil {
		if identity.Service() != nil {
			return nil
		}
		msg.SetUser(utils.IfThenElse(a.withoutPermissions,
			stripPermissionsFromUserInfo(identity.User()),
			identity.User(),
		))
	}

	var method, endpoint string
	if ri.HTTPRequest != nil {
		method = ri.HTTPRequest.Method
		endpoint = ri.HTTPRequest.URL.String()
		if referer := ri.HTTPRequest.Headers.Get(refererKey); referer != "" {
			msg.SetMethod(v1.Audit_UI)
		} else {
			msg.SetMethod(v1.Audit_API)
		}
	} else {
		method = defaultGRPCMethod
		endpoint = grpcFullMethod
		msg.SetMethod(v1.Audit_CLI)
	}

	interaction, ok := requestInteractionMap[method]
	if !ok {
		return nil
	}
	msg.SetInteraction(interaction)

	amrs := &v1.Audit_Message_Request_SourceHeaders{}
	amrs.SetXForwardedFor(ri.Source.XForwardedFor)
	amrs.SetRemoteAddr(ri.Source.RemoteAddr)
	amrs.SetRequestAddr(ri.Source.RequestAddr)
	amr := &v1.Audit_Message_Request{}
	amr.SetEndpoint(endpoint)
	amr.SetMethod(method)
	amr.SetPayload(protoutils.RequestToAny(req))
	amr.SetSourceHeaders(amrs)
	amr.SetSourceIp(ri.Source.GetSourceIP())
	msg.SetRequest(amr)

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

func stripPermissionsFromUserInfo(userInfo *storage.UserInfo) *storage.UserInfo {
	userInfoWithoutPermissions := userInfo.CloneVT()
	userInfoWithoutPermissions.ClearPermissions()

	userRolesWithoutPermissions := make([]*storage.UserInfo_Role, 0, len(userInfo.GetRoles()))
	for _, userRole := range userInfo.GetRoles() {
		ur := &storage.UserInfo_Role{}
		ur.SetName(userRole.GetName())
		userRolesWithoutPermissions = append(userRolesWithoutPermissions, ur)
	}
	userInfoWithoutPermissions.SetRoles(userRolesWithoutPermissions)

	return userInfoWithoutPermissions
}
