package testutils

import (
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	v2 "github.com/stackrox/rox/generated/api/v2"
)

var (
	ts1 = timestamp.TimestampNow()
)

// GetTestVulnDeferralExceptionFull returns a mock *v2.VulnerabilityException of deferral kind.
func GetTestVulnDeferralExceptionFull(_ *testing.T) *v2.VulnerabilityException {
	return &v2.VulnerabilityException{
		Id:          "id",
		Name:        "name",
		TargetState: v2.VulnerabilityState_DEFERRED,
		Status:      v2.ExceptionStatus_PENDING,
		Expired:     false,
		Requester: &v2.SlimUser{
			Id:   "userID",
			Name: "userName",
		},
		Approvers: []*v2.SlimUser{
			{
				Id:   "userID",
				Name: "userName",
			},
		},
		Comments: []*v2.Comment{
			{
				Id:      "commentID",
				Message: "message",
				User: &v2.SlimUser{
					Id:   "userID",
					Name: "userName",
				},
			},
		},
		Req: &v2.VulnerabilityException_DeferralReq{
			DeferralReq: &v2.DeferralRequest{
				Expiry: &v2.ExceptionExpiry{
					ExpiryType: v2.ExceptionExpiry_TIME,
					ExpiresOn:  ts1,
				},
			},
		},
		Scope: &v2.VulnerabilityException_Scope{
			ImageScope: &v2.VulnerabilityException_Scope_Image{
				Registry: "reg",
				Remote:   "remote",
				Tag:      "tag",
			},
		},
		Cves: []string{"cve1"},
	}
}

// GetTestVulnFPExceptionFull returns a mock *v2.VulnerabilityException of false-positive kind.
func GetTestVulnFPExceptionFull(t *testing.T) *v2.VulnerabilityException {
	ret := GetTestVulnDeferralExceptionFull(t)
	ret.TargetState = v2.VulnerabilityState_FALSE_POSITIVE
	ret.Req = &v2.VulnerabilityException_FpRequest{
		FpRequest: &v2.FalsePositiveRequest{},
	}
	return ret
}

// GetTestVulnExceptionNoUsers returns a mock *v2.VulnerabilityException with nil `.requester` and `.approvers` fields.
func GetTestVulnExceptionNoUsers(t *testing.T) *v2.VulnerabilityException {
	ret := GetTestVulnDeferralExceptionFull(t)
	ret.Requester = nil
	ret.Approvers = nil
	return ret
}

// GetTestVulnExceptionNoComments returns a mock *v2.VulnerabilityException with nil `.comments` field.
func GetTestVulnExceptionNoComments(t *testing.T) *v2.VulnerabilityException {
	ret := GetTestVulnDeferralExceptionFull(t)
	ret.Comments = nil
	return ret
}

// GetTestVulnExceptionWithUpdate returns a mock *v2.VulnerabilityException with non-nil `.updateReq` field.
func GetTestVulnExceptionWithUpdate(t *testing.T) *v2.VulnerabilityException {
	ret := GetTestVulnDeferralExceptionFull(t)
	ret.UpdatedReq = &v2.VulnerabilityException_DeferralReqUpdate{
		DeferralReqUpdate: &v2.DeferralRequest{
			Expiry: &v2.ExceptionExpiry{
				ExpiryType: v2.ExceptionExpiry_TIME,
				ExpiresOn:  ts1,
			},
		},
	}
	return ret
}

// GetTestCreateDeferVulnExceptionRequest returns a mock *v2.CreateDeferVulnerabilityExceptionRequest.
func GetTestCreateDeferVulnExceptionRequest(t *testing.T) *v2.CreateDeferVulnerabilityExceptionRequest {
	req := GetTestVulnDeferralExceptionFull(t)
	return &v2.CreateDeferVulnerabilityExceptionRequest{
		Cves:            []string{"cve1"},
		Comment:         "message",
		Scope:           req.GetScope(),
		ExceptionExpiry: req.GetDeferralReq().GetExpiry(),
	}
}

// GetTestCreateFPVulnExceptionRequest returns a mock *v2.CreateFalsePositiveVulnerabilityExceptionRequest.
func GetTestCreateFPVulnExceptionRequest(t *testing.T) *v2.CreateFalsePositiveVulnerabilityExceptionRequest {
	return &v2.CreateFalsePositiveVulnerabilityExceptionRequest{
		Cves:    []string{"cve1"},
		Comment: "message",
		Scope:   GetTestVulnDeferralExceptionFull(t).GetScope(),
	}
}
