package storagetov2

import (
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// VulnerabilityRequest converts *v2.VulnerabilityException to *storage.VulnerabilityRequest.
func VulnerabilityRequest(vulnException *v2.VulnerabilityException) *storage.VulnerabilityRequest {
	if vulnException == nil {
		return nil
	}

	out := &storage.VulnerabilityRequest{
		Id:          vulnException.GetId(),
		Name:        vulnException.GetName(),
		TargetState: convertVulnerabilityState(vulnException.GetTargetState()),
		Status:      convertRequestStatus(vulnException.GetStatus()),
		Expired:     vulnException.GetExpired(),
		Requestor:   convertUser(vulnException.GetRequester()),
		Approvers:   convertUsers(vulnException.GetApprovers()),
		LastUpdated: vulnException.GetLastUpdated(),
		Comments:    convertRequestComments(vulnException.GetComments()),
		Scope:       convertScope(vulnException.GetScope()),
		Entities: &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Cves: vulnException.GetCves(),
			},
		},
		UpdatedReq: nil,
	}

	if vulnException.GetDeferralReq() != nil {
		out.Req = &storage.VulnerabilityRequest_DeferralReq{
			DeferralReq: convertDeferralReq(vulnException.GetDeferralReq()),
		}
	} else if vulnException.GetFpRequest() != nil {
		out.Req = &storage.VulnerabilityRequest_FpRequest{
			FpRequest: &storage.FalsePositiveRequest{},
		}
	}

	if vulnException.GetDeferralReqUpdate() != nil {
		out.UpdatedReq = &storage.VulnerabilityRequest_UpdatedDeferralReq{
			UpdatedDeferralReq: convertDeferralReq(vulnException.GetDeferralReqUpdate()),
		}
	}

	return out
}

func convertRequestStatus(status v2.ExceptionStatus) storage.RequestStatus {
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

func convertRequestComments(comments []*v2.Comment) []*storage.RequestComment {
	if len(comments) == 0 {
		return nil
	}

	var ret []*storage.RequestComment
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		ret = append(ret, &storage.RequestComment{
			Id:        comment.GetId(),
			Message:   comment.GetMessage(),
			User:      convertUser(comment.GetUser()),
			CreatedAt: comment.GetCreatedAt(),
		})
	}
	return ret
}

func convertScope(scope *v2.VulnerabilityException_Scope) *storage.VulnerabilityRequest_Scope {
	if scope == nil || scope.GetImageScope() == nil {
		return nil
	}

	return &storage.VulnerabilityRequest_Scope{
		Info: &storage.VulnerabilityRequest_Scope_ImageScope{
			ImageScope: &storage.VulnerabilityRequest_Scope_Image{
				Registry: scope.GetImageScope().GetRegistry(),
				Remote:   scope.GetImageScope().GetRemote(),
				Tag:      scope.GetImageScope().GetTag(),
			},
		},
	}
}

func convertDeferralReq(r *v2.DeferralRequest) *storage.DeferralRequest {
	if r == nil {
		return nil
	}
	return &storage.DeferralRequest{
		Expiry: convertRequestExpiry(r.GetExpiry()),
	}
}

func convertRequestExpiry(expiry *v2.ExceptionExpiry) *storage.RequestExpiry {
	ret := &storage.RequestExpiry{
		ExpiryType: convertExpiryType(expiry.GetExpiryType()),
	}
	if expiry.GetExpiryType() == v2.ExceptionExpiry_TIME {
		ret.Expiry = &storage.RequestExpiry_ExpiresOn{
			ExpiresOn: expiry.GetExpiresOn(),
		}
	} else {
		ret.Expiry = &storage.RequestExpiry_ExpiresWhenFixed{
			ExpiresWhenFixed: true,
		}
	}
	return ret
}

func convertExpiryType(t v2.ExceptionExpiry_ExpiryType) storage.RequestExpiry_ExpiryType {
	switch t {
	case v2.ExceptionExpiry_TIME:
		return storage.RequestExpiry_TIME
	case v2.ExceptionExpiry_ALL_CVE_FIXABLE:
		return storage.RequestExpiry_ALL_CVE_FIXABLE
	case v2.ExceptionExpiry_ANY_CVE_FIXABLE:
		return storage.RequestExpiry_ANY_CVE_FIXABLE
	default:
		utils.Should(errors.Errorf("unhandled expiry type encountered %s", t))
		return storage.RequestExpiry_TIME
	}
}
