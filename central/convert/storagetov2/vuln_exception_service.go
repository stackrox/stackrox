package storagetov2

import (
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
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

	out := &v2.VulnerabilityException{
		Id:          vulnRequest.GetId(),
		Name:        vulnRequest.GetName(),
		TargetState: convertVulnerabilityState(vulnRequest.GetTargetState()),
		Status:      convertRequestStatus(vulnRequest.GetStatus()),
		Expired:     vulnRequest.GetExpired(),
		Requester:   convertUser(vulnRequest.GetRequestor()),
		Approvers:   convertUsers(vulnRequest.GetApprovers()),
		LastUpdated: vulnRequest.GetLastUpdated(),
		Comments:    convertRequestComments(vulnRequest.GetComments()),
		Scope:       convertScope(vulnRequest.GetScope()),
		Cves:        vulnRequest.GetCves().GetCves(),
	}

	if vulnRequest.GetDeferralReq() != nil {
		out.Req = &v2.VulnerabilityException_DeferralReq{
			DeferralReq: convertDeferralReq(vulnRequest.GetDeferralReq()),
		}
	} else if vulnRequest.GetFpRequest() != nil {
		out.Req = &v2.VulnerabilityException_FpRequest{
			FpRequest: &v2.FalsePositiveRequest{},
		}
	}

	if vulnRequest.GetUpdatedDeferralReq() != nil {
		out.UpdatedReq = &v2.VulnerabilityException_DeferralReqUpdate{
			DeferralReqUpdate: convertDeferralReq(vulnRequest.GetUpdatedDeferralReq()),
		}
	}

	return out
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
