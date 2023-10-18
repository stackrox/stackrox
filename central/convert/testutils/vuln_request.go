package testutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

// GetTestVulnDeferralRequestFull returns a mock *storage.VulnerabilityRequest of deferral kind.
func GetTestVulnDeferralRequestFull(_ *testing.T) *storage.VulnerabilityRequest {
	return &storage.VulnerabilityRequest{
		Id:          "id",
		Name:        "name",
		TargetState: storage.VulnerabilityState_DEFERRED,
		Status:      storage.RequestStatus_PENDING,
		Expired:     false,
		Requestor: &storage.SlimUser{
			Id:   "userID",
			Name: "userName",
		},
		Approvers: []*storage.SlimUser{
			{
				Id:   "userID",
				Name: "userName",
			},
		},
		Comments: []*storage.RequestComment{
			{
				Id:      "commentID",
				Message: "message",
				User: &storage.SlimUser{
					Id:   "userID",
					Name: "userName",
				},
			},
		},
		Req: &storage.VulnerabilityRequest_DeferralReq{
			DeferralReq: &storage.DeferralRequest{
				Expiry: &storage.RequestExpiry{
					ExpiryType: storage.RequestExpiry_TIME,
					Expiry: &storage.RequestExpiry_ExpiresOn{
						ExpiresOn: ts1,
					},
				},
			},
		},
		Scope: &storage.VulnerabilityRequest_Scope{
			Info: &storage.VulnerabilityRequest_Scope_ImageScope{
				ImageScope: &storage.VulnerabilityRequest_Scope_Image{
					Registry: "reg",
					Remote:   "remote",
					Tag:      "tag",
				},
			},
		},
		Entities: &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Cves: []string{"cve1"},
			},
		},
	}
}

// GetTestVulnFPRequestFull returns a mock *storage.VulnerabilityRequest of false-positive kind.
func GetTestVulnFPRequestFull(t *testing.T) *storage.VulnerabilityRequest {
	ret := GetTestVulnDeferralRequestFull(t)
	ret.TargetState = storage.VulnerabilityState_FALSE_POSITIVE
	ret.Req = &storage.VulnerabilityRequest_FpRequest{
		FpRequest: &storage.FalsePositiveRequest{},
	}
	return ret
}

// GetTestVulnRequestNoUsers returns a mock *storage.VulnerabilityRequest with nil `.requester` and `.approvers` fields.
func GetTestVulnRequestNoUsers(t *testing.T) *storage.VulnerabilityRequest {
	ret := GetTestVulnDeferralRequestFull(t)
	ret.Requestor = nil
	ret.Approvers = nil
	return ret
}

// GetTestVulnRequestNoComments returns a mock *storage.VulnerabilityRequest with nil `.comments` field.
func GetTestVulnRequestNoComments(t *testing.T) *storage.VulnerabilityRequest {
	ret := GetTestVulnDeferralRequestFull(t)
	ret.Comments = nil
	return ret
}

// GetTestVulnRequestWithUpdate returns a mock *storage.VulnerabilityRequest with non-nil `.updateReq` field.
func GetTestVulnRequestWithUpdate(t *testing.T) *storage.VulnerabilityRequest {
	ret := GetTestVulnDeferralRequestFull(t)
	ret.UpdatedReq = &storage.VulnerabilityRequest_UpdatedDeferralReq{
		UpdatedDeferralReq: &storage.DeferralRequest{
			Expiry: &storage.RequestExpiry{
				ExpiryType: storage.RequestExpiry_TIME,
				Expiry: &storage.RequestExpiry_ExpiresOn{
					ExpiresOn: ts1,
				},
			},
		},
	}
	return ret
}
