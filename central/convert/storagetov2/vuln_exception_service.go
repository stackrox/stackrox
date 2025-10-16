package storagetov2

import (
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/proto"
)

// VulnerabilityExceptions converts a slice of *storage.VulnerabilityRequest to a slice of *v2.VulnerabilityException.
func VulnerabilityExceptions(inp ...*storage.VulnerabilityRequest) []*v2.VulnerabilityException {
	ret := make([]*v2.VulnerabilityException, 0, len(inp))
	for _, obj := range inp {
		if obj == nil {
			continue
		}
		ret = append(ret, VulnerabilityException(obj))
	}
	return ret
}

// VulnerabilityException converts *storage.VulnerabilityRequest to *v2.VulnerabilityException.
func VulnerabilityException(vulnRequest *storage.VulnerabilityRequest) *v2.VulnerabilityException {
	if vulnRequest == nil {
		return nil
	}

	out := &v2.VulnerabilityException{}
	out.SetId(vulnRequest.GetId())
	out.SetName(vulnRequest.GetName())
	out.SetTargetState(convertVulnerabilityState(vulnRequest.GetTargetState()))
	out.SetStatus(exceptionStatus(vulnRequest.GetStatus()))
	out.SetExpired(vulnRequest.GetExpired())
	out.SetRequester(requester(vulnRequest.GetRequesterV2()))
	out.SetApprovers(approvers(vulnRequest.GetApproversV2()))
	out.SetCreatedAt(vulnRequest.GetCreatedAt())
	out.SetLastUpdated(vulnRequest.GetLastUpdated())
	out.SetComments(comments(vulnRequest.GetComments()))
	out.SetScope(exceptionScope(vulnRequest.GetScope()))
	out.SetCves(vulnRequest.GetCves().GetCves())

	if vulnRequest.GetDeferralReq() != nil {
		out.SetDeferralRequest(proto.ValueOrDefault(deferralRequest(vulnRequest.GetDeferralReq())))
	} else if vulnRequest.GetFpRequest() != nil {
		out.SetFalsePositiveRequest(&v2.FalsePositiveRequest{})
	}

	if vulnRequest.GetDeferralUpdate() != nil {
		out.SetDeferralUpdate(proto.ValueOrDefault(deferralUpdate(vulnRequest.GetDeferralUpdate())))
	} else if vulnRequest.GetFalsePositiveUpdate() != nil {
		out.SetFalsePositiveUpdate(proto.ValueOrDefault(falsePositiveUpdate(vulnRequest.GetFalsePositiveUpdate())))
	}

	return out
}

func exceptionStatus(status storage.RequestStatus) v2.ExceptionStatus {
	switch status {
	case storage.RequestStatus_PENDING:
		return v2.ExceptionStatus_PENDING
	case storage.RequestStatus_APPROVED:
		return v2.ExceptionStatus_APPROVED
	case storage.RequestStatus_DENIED:
		return v2.ExceptionStatus_DENIED
	case storage.RequestStatus_APPROVED_PENDING_UPDATE:
		return v2.ExceptionStatus_APPROVED_PENDING_UPDATE
	default:
		utils.Should(errors.Errorf("unhandled request status encountered %s", status))
		return v2.ExceptionStatus_PENDING
	}
}

func comments(comments []*storage.RequestComment) []*v2.Comment {
	if len(comments) == 0 {
		return nil
	}

	var ret []*v2.Comment
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		comment2 := &v2.Comment{}
		comment2.SetId(comment.GetId())
		comment2.SetMessage(comment.GetMessage())
		comment2.SetUser(convertUser(comment.GetUser()))
		comment2.SetCreatedAt(comment.GetCreatedAt())
		ret = append(ret, comment2)
	}
	return ret
}

func exceptionScope(scope *storage.VulnerabilityRequest_Scope) *v2.VulnerabilityException_Scope {
	if scope == nil || scope.GetImageScope() == nil {
		return nil
	}

	vsi := &v2.VulnerabilityException_Scope_Image{}
	vsi.SetRegistry(scope.GetImageScope().GetRegistry())
	vsi.SetRemote(scope.GetImageScope().GetRemote())
	vsi.SetTag(scope.GetImageScope().GetTag())
	vs := &v2.VulnerabilityException_Scope{}
	vs.SetImageScope(vsi)
	return vs
}

func deferralRequest(r *storage.DeferralRequest) *v2.DeferralRequest {
	if r == nil {
		return nil
	}
	dr := &v2.DeferralRequest{}
	dr.SetExpiry(exceptionExpiry(r.GetExpiry()))
	return dr
}

func exceptionExpiry(expiry *storage.RequestExpiry) *v2.ExceptionExpiry {
	ee := &v2.ExceptionExpiry{}
	ee.SetExpiryType(exceptionExpiryType(expiry.GetExpiryType()))
	ee.SetExpiresOn(expiry.GetExpiresOn())
	return ee
}

func exceptionExpiryType(t storage.RequestExpiry_ExpiryType) v2.ExceptionExpiry_ExpiryType {
	switch t {
	case storage.RequestExpiry_TIME:
		return v2.ExceptionExpiry_TIME
	case storage.RequestExpiry_ALL_CVE_FIXABLE:
		return v2.ExceptionExpiry_ALL_CVE_FIXABLE
	case storage.RequestExpiry_ANY_CVE_FIXABLE:
		return v2.ExceptionExpiry_ANY_CVE_FIXABLE
	default:
		utils.Should(errors.Errorf("unhandled expiry type encountered %s", t))
		return v2.ExceptionExpiry_TIME
	}
}

func deferralUpdate(update *storage.DeferralUpdate) *v2.DeferralUpdate {
	du := &v2.DeferralUpdate{}
	du.SetCves(update.GetCVEs())
	du.SetExpiry(exceptionExpiry(update.GetExpiry()))
	return du
}

func falsePositiveUpdate(update *storage.FalsePositiveUpdate) *v2.FalsePositiveUpdate {
	fpu := &v2.FalsePositiveUpdate{}
	fpu.SetCves(update.GetCVEs())
	return fpu
}

func requester(user *storage.Requester) *v2.SlimUser {
	if user == nil {
		return nil
	}
	slimUser := &v2.SlimUser{}
	slimUser.SetId(user.GetId())
	slimUser.SetName(user.GetName())
	return slimUser
}

func approvers(users []*storage.Approver) []*v2.SlimUser {
	ret := make([]*v2.SlimUser, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		slimUser := &v2.SlimUser{}
		slimUser.SetId(user.GetId())
		slimUser.SetName(user.GetName())
		ret = append(ret, slimUser)
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}
