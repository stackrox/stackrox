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
		Status:      exceptionStatus(vulnRequest.GetStatus()),
		Expired:     vulnRequest.GetExpired(),
		Requester:   requester(vulnRequest.GetRequesterV2()),
		Approvers:   approvers(vulnRequest.GetApproversV2()),
		CreatedAt:   vulnRequest.GetCreatedAt(),
		LastUpdated: vulnRequest.GetLastUpdated(),
		Comments:    comments(vulnRequest.GetComments()),
		Scope:       exceptionScope(vulnRequest.GetScope()),
		Cves:        vulnRequest.GetCves().GetCves(),
	}

	if vulnRequest.GetDeferralReq() != nil {
		out.Req = &v2.VulnerabilityException_DeferralRequest{
			DeferralRequest: deferralRequest(vulnRequest.GetDeferralReq()),
		}
	} else if vulnRequest.GetFpRequest() != nil {
		out.Req = &v2.VulnerabilityException_FalsePositiveRequest{
			FalsePositiveRequest: &v2.FalsePositiveRequest{},
		}
	}

	if vulnRequest.GetDeferralUpdate() != nil {
		out.UpdatedReq = &v2.VulnerabilityException_DeferralUpdate{
			DeferralUpdate: deferralUpdate(vulnRequest.GetDeferralUpdate()),
		}
	} else if vulnRequest.GetFalsePositiveUpdate() != nil {
		out.UpdatedReq = &v2.VulnerabilityException_FalsePositiveUpdate{
			FalsePositiveUpdate: falsePositiveUpdate(vulnRequest.GetFalsePositiveUpdate()),
		}
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
		ret = append(ret, &v2.Comment{
			Id:        comment.GetId(),
			Message:   comment.GetMessage(),
			User:      convertUser(comment.GetUser()),
			CreatedAt: comment.GetCreatedAt(),
		})
	}
	return ret
}

func exceptionScope(scope *storage.VulnerabilityRequest_Scope) *v2.VulnerabilityException_Scope {
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

func deferralRequest(r *storage.DeferralRequest) *v2.DeferralRequest {
	if r == nil {
		return nil
	}
	return &v2.DeferralRequest{
		Expiry: exceptionExpiry(r.GetExpiry()),
	}
}

func exceptionExpiry(expiry *storage.RequestExpiry) *v2.ExceptionExpiry {
	return &v2.ExceptionExpiry{
		ExpiryType: exceptionExpiryType(expiry.GetExpiryType()),
		ExpiresOn:  expiry.GetExpiresOn(),
	}
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
	return &v2.DeferralUpdate{
		Cves:   update.GetCVEs(),
		Expiry: exceptionExpiry(update.GetExpiry()),
	}
}

func falsePositiveUpdate(update *storage.FalsePositiveUpdate) *v2.FalsePositiveUpdate {
	return &v2.FalsePositiveUpdate{
		CVEs: update.GetCVEs(),
	}
}

func requester(user *storage.Requester) *v2.SlimUser {
	if user == nil {
		return nil
	}
	return &v2.SlimUser{
		Id:   user.GetId(),
		Name: user.GetName(),
	}
}

func approvers(users []*storage.Approver) []*v2.SlimUser {
	ret := make([]*v2.SlimUser, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		ret = append(ret, &v2.SlimUser{
			Id:   user.GetId(),
			Name: user.GetName(),
		})
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}
