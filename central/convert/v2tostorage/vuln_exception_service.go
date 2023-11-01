package v2tostorage

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
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
		Status:      requestStatus(vulnException.GetStatus()),
		Expired:     vulnException.GetExpired(),
		// Fill the legacy field for backward compatibility.
		Requestor: convertUser(vulnException.GetRequester()),
		// Fill the legacy field for backward compatibility.
		Approvers:   convertUsers(vulnException.GetApprovers()),
		LastUpdated: vulnException.GetLastUpdated(),
		Comments:    requestComments(vulnException.GetComments()),
		Scope:       requestScope(vulnException.GetScope()),
		Entities: &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Cves: vulnException.GetCves(),
			},
		},
		UpdatedReq: nil,
	}
	out.RequesterV2 = requester(out.GetRequestor())
	out.ApproversV2 = approvers(out.GetApprovers())

	if vulnException.GetDeferralRequest() != nil {
		out.Req = &storage.VulnerabilityRequest_DeferralReq{
			DeferralReq: deferralRequest(vulnException.GetDeferralRequest()),
		}
	} else if vulnException.GetFalsePositiveRequest() != nil {
		out.Req = &storage.VulnerabilityRequest_FpRequest{
			FpRequest: &storage.FalsePositiveRequest{},
		}
	}

	if vulnException.GetDeferralUpdate() != nil {
		out.UpdatedReq = &storage.VulnerabilityRequest_DeferralUpdate{
			DeferralUpdate: DeferralUpdate(vulnException.GetDeferralUpdate()),
		}
	} else if vulnException.GetFalsePositiveUpdate() != nil {
		out.UpdatedReq = &storage.VulnerabilityRequest_FalsePositiveUpdate{
			FalsePositiveUpdate: FalsePositiveUpdate(vulnException.GetFalsePositiveUpdate()),
		}
	}
	return out
}

// DeferVulnerabilityRequest converts a *v2.CreateDeferVulnerabilityExceptionRequest to a *storage.VulnerabilityRequest.
func DeferVulnerabilityRequest(ctx context.Context, req *v2.CreateDeferVulnerabilityExceptionRequest) *storage.VulnerabilityRequest {
	now := types.TimestampNow()
	ret := &storage.VulnerabilityRequest{
		CreatedAt:   now,
		LastUpdated: now,
		TargetState: storage.VulnerabilityState_DEFERRED,
		Status:      storage.RequestStatus_PENDING,
		Requestor:   authn.UserFromContext(ctx),
		Scope:       requestScope(req.GetScope()),
	}
	ret.RequesterV2 = requester(ret.GetRequestor())
	if req.GetExceptionExpiry() != nil {
		ret.Req = &storage.VulnerabilityRequest_DeferralReq{
			DeferralReq: &storage.DeferralRequest{
				Expiry: requestExpiry(req.GetExceptionExpiry()),
			},
		}
	}
	if len(req.GetCves()) > 0 {
		ret.Entities = &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Cves: req.GetCves(),
			},
		}
	}
	if comment := req.GetComment(); comment != "" {
		ret.Comments = []*storage.RequestComment{
			{
				Id:        uuid.NewV4().String(),
				CreatedAt: now,
				Message:   comment,
				User:      authn.UserFromContext(ctx),
			},
		}
	}
	return ret
}

// FalsePositiveVulnerabilityRequest converts a *v2.CreateFalsePositiveVulnerabilityExceptionRequest to a *storage.VulnerabilityRequest.
func FalsePositiveVulnerabilityRequest(ctx context.Context, req *v2.CreateFalsePositiveVulnerabilityExceptionRequest) *storage.VulnerabilityRequest {
	now := types.TimestampNow()
	ret := &storage.VulnerabilityRequest{
		CreatedAt:   now,
		LastUpdated: now,
		TargetState: storage.VulnerabilityState_FALSE_POSITIVE,
		Status:      storage.RequestStatus_PENDING,
		Requestor:   authn.UserFromContext(ctx),
		Req: &storage.VulnerabilityRequest_FpRequest{
			FpRequest: &storage.FalsePositiveRequest{},
		},
		Scope: requestScope(req.GetScope()),
	}
	ret.RequesterV2 = requester(ret.GetRequestor())
	if len(req.GetCves()) > 0 {
		ret.Entities = &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Cves: req.GetCves(),
			},
		}
	}
	if comment := req.GetComment(); comment != "" {
		ret.Comments = []*storage.RequestComment{
			{
				Id:        uuid.NewV4().String(),
				CreatedAt: now,
				Message:   comment,
				User:      authn.UserFromContext(ctx),
			},
		}
	}
	return ret
}

// DeferralUpdate converts *v2.DeferralUpdate object to *storage.DeferralUpdate object.
func DeferralUpdate(update *v2.DeferralUpdate) *storage.DeferralUpdate {
	return &storage.DeferralUpdate{
		CVEs:   update.GetCves(),
		Expiry: requestExpiry(update.GetExpiry()),
	}
}

// FalsePositiveUpdate converts *v2.FalsePositiveUpdate object to  *storage.FalsePositiveUpdate.
func FalsePositiveUpdate(update *v2.FalsePositiveUpdate) *storage.FalsePositiveUpdate {
	return &storage.FalsePositiveUpdate{
		CVEs: update.GetCVEs(),
	}
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
		ret = append(ret, &storage.RequestComment{
			Id:        comment.GetId(),
			Message:   comment.GetMessage(),
			User:      convertUser(comment.GetUser()),
			CreatedAt: comment.GetCreatedAt(),
		})
	}
	return ret
}

func requestScope(scope *v2.VulnerabilityException_Scope) *storage.VulnerabilityRequest_Scope {
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

func deferralRequest(r *v2.DeferralRequest) *storage.DeferralRequest {
	if r == nil {
		return nil
	}
	return &storage.DeferralRequest{
		Expiry: requestExpiry(r.GetExpiry()),
	}
}

func requestExpiry(expiry *v2.ExceptionExpiry) *storage.RequestExpiry {
	ret := &storage.RequestExpiry{
		ExpiryType: requestExpiryType(expiry.GetExpiryType()),
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
	return &storage.Requester{
		Id:   user.GetId(),
		Name: user.GetName(),
	}
}

func approvers(users []*storage.SlimUser) []*storage.Approver {
	ret := make([]*storage.Approver, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		ret = append(ret, &storage.Approver{
			Id:   user.GetId(),
			Name: user.GetName(),
		})
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}
