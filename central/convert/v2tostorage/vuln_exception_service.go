package v2tostorage

import (
	"context"

	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
)

// VulnerabilityRequest converts *v2.VulnerabilityException to *storage.VulnerabilityRequest.
func VulnerabilityRequest(vulnException *v2.VulnerabilityException) *storage.VulnerabilityRequest {
	if vulnException == nil {
		return nil
	}

	vc := &storage.VulnerabilityRequest_CVEs{}
	vc.SetCves(vulnException.GetCves())
	out := &storage.VulnerabilityRequest{}
	out.SetId(vulnException.GetId())
	out.SetName(vulnException.GetName())
	out.SetTargetState(convertVulnerabilityState(vulnException.GetTargetState()))
	out.SetStatus(requestStatus(vulnException.GetStatus()))
	out.SetExpired(vulnException.GetExpired())
	// Fill the legacy field for backward compatibility.
	out.SetRequestor(convertUser(vulnException.GetRequester()))
	// Fill the legacy field for backward compatibility.
	out.SetApprovers(convertUsers(vulnException.GetApprovers()))
	out.SetLastUpdated(vulnException.GetLastUpdated())
	out.SetComments(requestComments(vulnException.GetComments()))
	out.SetScope(requestScope(vulnException.GetScope()))
	out.SetCves(proto.ValueOrDefault(vc))
	out.ClearUpdatedReq()
	out.SetRequesterV2(requester(out.GetRequestor()))
	out.SetApproversV2(approvers(out.GetApprovers()))

	if vulnException.GetDeferralRequest() != nil {
		out.SetDeferralReq(proto.ValueOrDefault(deferralRequest(vulnException.GetDeferralRequest())))
	} else if vulnException.GetFalsePositiveRequest() != nil {
		out.SetFpRequest(&storage.FalsePositiveRequest{})
	}

	if vulnException.GetDeferralUpdate() != nil {
		out.SetDeferralUpdate(proto.ValueOrDefault(DeferralUpdate(vulnException.GetDeferralUpdate())))
	} else if vulnException.GetFalsePositiveUpdate() != nil {
		out.SetFalsePositiveUpdate(proto.ValueOrDefault(FalsePositiveUpdate(vulnException.GetFalsePositiveUpdate())))
	}
	return out
}

// DeferVulnerabilityRequest converts a *v2.CreateDeferVulnerabilityExceptionRequest to a *storage.VulnerabilityRequest.
func DeferVulnerabilityRequest(ctx context.Context, req *v2.CreateDeferVulnerabilityExceptionRequest) *storage.VulnerabilityRequest {
	now := protocompat.TimestampNow()
	ret := &storage.VulnerabilityRequest{}
	ret.SetCreatedAt(now)
	ret.SetLastUpdated(now)
	ret.SetTargetState(storage.VulnerabilityState_DEFERRED)
	ret.SetStatus(storage.RequestStatus_PENDING)
	ret.SetRequestor(authn.UserFromContext(ctx))
	ret.SetScope(requestScope(req.GetScope()))
	ret.SetRequesterV2(requester(ret.GetRequestor()))
	if req.GetExceptionExpiry() != nil {
		dr := &storage.DeferralRequest{}
		dr.SetExpiry(requestExpiry(req.GetExceptionExpiry()))
		ret.SetDeferralReq(proto.ValueOrDefault(dr))
	}
	if len(req.GetCves()) > 0 {
		vc := &storage.VulnerabilityRequest_CVEs{}
		vc.SetCves(req.GetCves())
		ret.SetCves(proto.ValueOrDefault(vc))
	}
	if comment := req.GetComment(); comment != "" {
		rc := &storage.RequestComment{}
		rc.SetId(uuid.NewV4().String())
		rc.SetCreatedAt(now)
		rc.SetMessage(comment)
		rc.SetUser(authn.UserFromContext(ctx))
		ret.SetComments([]*storage.RequestComment{
			rc,
		})
	}
	return ret
}

// FalsePositiveVulnerabilityRequest converts a *v2.CreateFalsePositiveVulnerabilityExceptionRequest to a *storage.VulnerabilityRequest.
func FalsePositiveVulnerabilityRequest(ctx context.Context, req *v2.CreateFalsePositiveVulnerabilityExceptionRequest) *storage.VulnerabilityRequest {
	now := protocompat.TimestampNow()
	ret := &storage.VulnerabilityRequest{}
	ret.SetCreatedAt(now)
	ret.SetLastUpdated(now)
	ret.SetTargetState(storage.VulnerabilityState_FALSE_POSITIVE)
	ret.SetStatus(storage.RequestStatus_PENDING)
	ret.SetRequestor(authn.UserFromContext(ctx))
	ret.SetFpRequest(&storage.FalsePositiveRequest{})
	ret.SetScope(requestScope(req.GetScope()))
	ret.SetRequesterV2(requester(ret.GetRequestor()))
	if len(req.GetCves()) > 0 {
		vc := &storage.VulnerabilityRequest_CVEs{}
		vc.SetCves(req.GetCves())
		ret.SetCves(proto.ValueOrDefault(vc))
	}
	if comment := req.GetComment(); comment != "" {
		rc := &storage.RequestComment{}
		rc.SetId(uuid.NewV4().String())
		rc.SetCreatedAt(now)
		rc.SetMessage(comment)
		rc.SetUser(authn.UserFromContext(ctx))
		ret.SetComments([]*storage.RequestComment{
			rc,
		})
	}
	return ret
}

// DeferralUpdate converts *v2.DeferralUpdate object to *storage.DeferralUpdate object.
func DeferralUpdate(update *v2.DeferralUpdate) *storage.DeferralUpdate {
	du := &storage.DeferralUpdate{}
	du.SetCVEs(update.GetCves())
	du.SetExpiry(requestExpiry(update.GetExpiry()))
	return du
}

// FalsePositiveUpdate converts *v2.FalsePositiveUpdate object to  *storage.FalsePositiveUpdate.
func FalsePositiveUpdate(update *v2.FalsePositiveUpdate) *storage.FalsePositiveUpdate {
	fpu := &storage.FalsePositiveUpdate{}
	fpu.SetCVEs(update.GetCves())
	return fpu
}

func requestStatus(status v2.ExceptionStatus) storage.RequestStatus {
	switch status {
	case v2.ExceptionStatus_PENDING:
		return storage.RequestStatus_PENDING
	case v2.ExceptionStatus_APPROVED:
		return storage.RequestStatus_APPROVED
	case v2.ExceptionStatus_DENIED:
		return storage.RequestStatus_DENIED
	case v2.ExceptionStatus_APPROVED_PENDING_UPDATE:
		return storage.RequestStatus_APPROVED_PENDING_UPDATE
	default:
		utils.Should(errors.Errorf("unhandled request status encountered %s", status))
		return storage.RequestStatus_PENDING
	}
}

func requestComments(comments []*v2.Comment) []*storage.RequestComment {
	if len(comments) == 0 {
		return nil
	}

	var ret []*storage.RequestComment
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		rc := &storage.RequestComment{}
		rc.SetId(comment.GetId())
		rc.SetMessage(comment.GetMessage())
		rc.SetUser(convertUser(comment.GetUser()))
		rc.SetCreatedAt(comment.GetCreatedAt())
		ret = append(ret, rc)
	}
	return ret
}

func requestScope(scope *v2.VulnerabilityException_Scope) *storage.VulnerabilityRequest_Scope {
	if scope == nil || scope.GetImageScope() == nil {
		return nil
	}

	vsi := &storage.VulnerabilityRequest_Scope_Image{}
	vsi.SetRegistry(scope.GetImageScope().GetRegistry())
	vsi.SetRemote(scope.GetImageScope().GetRemote())
	vsi.SetTag(scope.GetImageScope().GetTag())
	vs := &storage.VulnerabilityRequest_Scope{}
	vs.SetImageScope(proto.ValueOrDefault(vsi))
	return vs
}

func deferralRequest(r *v2.DeferralRequest) *storage.DeferralRequest {
	if r == nil {
		return nil
	}
	dr := &storage.DeferralRequest{}
	dr.SetExpiry(requestExpiry(r.GetExpiry()))
	return dr
}

func requestExpiry(expiry *v2.ExceptionExpiry) *storage.RequestExpiry {
	ret := &storage.RequestExpiry{}
	ret.SetExpiryType(requestExpiryType(expiry.GetExpiryType()))
	switch expiry.GetExpiryType() {
	case v2.ExceptionExpiry_TIME:
		if expiry.GetExpiresOn() != nil {
			ret.SetExpiresOn(proto.ValueOrDefault(expiry.GetExpiresOn()))
		}
	case v2.ExceptionExpiry_ANY_CVE_FIXABLE:
		// Set the legacy field for backward compatibility.
		// In v1, a vulnerability request could have only one CVE at a time. For expiry based on CVE fixability,
		// the request expired if at least one CVE in the request was fixable which maps to ANY_CVE_FIXABLE behaviour in the v2.
		ret.SetExpiresWhenFixed(true)
	}
	return ret
}

func requestExpiryType(t v2.ExceptionExpiry_ExpiryType) storage.RequestExpiry_ExpiryType {
	switch t {
	case v2.ExceptionExpiry_TIME:
		return storage.RequestExpiry_TIME
	case v2.ExceptionExpiry_ALL_CVE_FIXABLE:
		return storage.RequestExpiry_ALL_CVE_FIXABLE
	case v2.ExceptionExpiry_ANY_CVE_FIXABLE:
		return storage.RequestExpiry_ANY_CVE_FIXABLE
	default:
		utils.Should(errors.Errorf("unhandled requestExpiry type encountered %s", t))
		return storage.RequestExpiry_TIME
	}
}

func requester(user *storage.SlimUser) *storage.Requester {
	if user == nil {
		return nil
	}
	requester2 := &storage.Requester{}
	requester2.SetId(user.GetId())
	requester2.SetName(user.GetName())
	return requester2
}

func approvers(users []*storage.SlimUser) []*storage.Approver {
	ret := make([]*storage.Approver, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		approver := &storage.Approver{}
		approver.SetId(user.GetId())
		approver.SetName(user.GetName())
		ret = append(ret, approver)
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}
