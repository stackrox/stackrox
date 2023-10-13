package storagetov2

import (
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// VulnerabilityException converts *storage.VulnerabilityRequest to *v2.VulnerabilityException.
func VulnerabilityException(inp *storage.VulnerabilityRequest) *v2.VulnerabilityException {
	if inp == nil {
		return nil
	}

	out := &v2.VulnerabilityException{
		Id:          inp.GetId(),
		Name:        inp.GetName(),
		TargetState: convertVulnerabilityState(inp.GetTargetState()),
		Status:      convertRequestStatus(inp.GetStatus()),
		Expired:     inp.GetExpired(),
		Requester:   convertUser(inp.GetRequestor()),
		Approvers:   convertUsers(inp.GetApprovers()),
		LastUpdated: inp.GetLastUpdated(),
		Comments:    convertRequestComments(inp.GetComments()),
		Scope:       convertScope(inp.GetScope()),
		Cves:        inp.GetCves().GetCves(),
	}

	if inp.GetDeferralReq() != nil {
		out.Req = &v2.VulnerabilityException_DeferralReq{
			DeferralReq: convertDeferralReq(inp.GetDeferralReq()),
		}
	} else if inp.GetFpRequest() != nil {
		out.Req = &v2.VulnerabilityException_FpRequest{
			FpRequest: &v2.FalsePositiveRequest{},
		}
	}

	if inp.GetUpdatedDeferralReq() != nil {
		out.UpdatedReq = &v2.VulnerabilityException_DeferralReqUpdate{
			DeferralReqUpdate: convertDeferralReq(inp.GetUpdatedDeferralReq()),
		}
	}

	return out
}

func convertUsers(users []*storage.SlimUser) []*v2.SlimUser {
	if len(users) == 0 {
		return nil
	}

	var ret []*v2.SlimUser
	for _, user := range users {
		if user == nil {
			continue
		}
		ret = append(ret, convertUser(user))
	}

	return ret
}

func convertUser(user *storage.SlimUser) *v2.SlimUser {
	if user == nil {
		return nil
	}

	return &v2.SlimUser{
		Id:   user.GetId(),
		Name: user.GetName(),
	}
}

func convertVulnerabilityState(state storage.VulnerabilityState) v2.VulnerabilityState {
	switch state {
	case storage.VulnerabilityState_OBSERVED:
		return v2.VulnerabilityState_OBSERVED
	case storage.VulnerabilityState_DEFERRED:
		return v2.VulnerabilityState_DEFERRED
	case storage.VulnerabilityState_FALSE_POSITIVE:
		return v2.VulnerabilityState_FALSE_POSITIVE
	default:
		utils.Should(errors.Errorf("unhandled vulnerability state encountered %s", state))
		return v2.VulnerabilityState_OBSERVED
	}
}

func convertRequestStatus(status storage.RequestStatus) v2.ExceptionStatus {
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

func convertRequestComments(comments []*storage.RequestComment) []*v2.Comment {
	if len(comments) == 0 {
		return nil
	}

	var ret []*v2.Comment
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		ret = append(ret, &v2.Comment{
			Id:        comment.GetId(),
			Message:   comment.GetMessage(),
			User:      convertUser(comment.GetUser()),
			CreatedAt: comment.GetCreatedAt(),
		})
	}
	return ret
}

func convertScope(scope *storage.VulnerabilityRequest_Scope) *v2.VulnerabilityException_Scope {
	if scope == nil || scope.GetImageScope() == nil {
		return nil
	}

	return &v2.VulnerabilityException_Scope{
		ImageScope: &v2.VulnerabilityException_Scope_Image{
			Registry: scope.GetImageScope().GetRegistry(),
			Remote:   scope.GetImageScope().GetRemote(),
			Tag:      scope.GetImageScope().GetTag(),
		},
	}
}

func convertDeferralReq(r *storage.DeferralRequest) *v2.DeferralRequest {
	if r == nil {
		return nil
	}
	return &v2.DeferralRequest{
		Expiry: convertRequestExpiry(r.GetExpiry()),
	}
}

func convertRequestExpiry(expiry *storage.RequestExpiry) *v2.ExceptionExpiry {
	return &v2.ExceptionExpiry{
		ExpiryType: convertExpiryType(expiry.GetExpiryType()),
		ExpiresOn:  expiry.GetExpiresOn(),
	}
}

func convertExpiryType(t storage.RequestExpiry_ExpiryType) v2.ExceptionExpiry_ExpiryType {
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
