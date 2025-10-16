package testutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/protobuf/proto"
)

// GetTestVulnDeferralRequestFull returns a mock *storage.VulnerabilityRequest of deferral kind.
func GetTestVulnDeferralRequestFull(_ *testing.T) *storage.VulnerabilityRequest {
	return storage.VulnerabilityRequest_builder{
		Id:          "id",
		Name:        "name",
		TargetState: storage.VulnerabilityState_DEFERRED,
		Status:      storage.RequestStatus_PENDING,
		Expired:     false,
		Requestor: storage.SlimUser_builder{
			Id:   "userID",
			Name: "userName",
		}.Build(),
		Approvers: []*storage.SlimUser{
			storage.SlimUser_builder{
				Id:   "userID",
				Name: "userName",
			}.Build(),
		},
		RequesterV2: storage.Requester_builder{
			Id:   "userID",
			Name: "userName",
		}.Build(),
		ApproversV2: []*storage.Approver{
			storage.Approver_builder{
				Id:   "userID",
				Name: "userName",
			}.Build(),
		},
		Comments: []*storage.RequestComment{
			storage.RequestComment_builder{
				Id:      "commentID",
				Message: "message",
				User: storage.SlimUser_builder{
					Id:   "userID",
					Name: "userName",
				}.Build(),
			}.Build(),
		},
		DeferralReq: storage.DeferralRequest_builder{
			Expiry: storage.RequestExpiry_builder{
				ExpiryType: storage.RequestExpiry_TIME,
				ExpiresOn:  proto.ValueOrDefault(ts1),
			}.Build(),
		}.Build(),
		Scope: storage.VulnerabilityRequest_Scope_builder{
			ImageScope: storage.VulnerabilityRequest_Scope_Image_builder{
				Registry: "reg",
				Remote:   "remote",
				Tag:      "tag",
			}.Build(),
		}.Build(),
		Cves: storage.VulnerabilityRequest_CVEs_builder{
			Cves: []string{"cve1"},
		}.Build(),
	}.Build()
}

// GetTestVulnFPRequestFull returns a mock *storage.VulnerabilityRequest of false-positive kind.
func GetTestVulnFPRequestFull(t *testing.T) *storage.VulnerabilityRequest {
	ret := GetTestVulnDeferralRequestFull(t)
	ret.SetTargetState(storage.VulnerabilityState_FALSE_POSITIVE)
	ret.SetFpRequest(&storage.FalsePositiveRequest{})
	return ret
}

// GetTestVulnRequestNoUsers returns a mock *storage.VulnerabilityRequest with nil `.requester` and `.approvers` fields.
func GetTestVulnRequestNoUsers(t *testing.T) *storage.VulnerabilityRequest {
	ret := GetTestVulnDeferralRequestFull(t)
	ret.ClearRequestor()
	ret.SetApprovers(nil)
	ret.ClearRequesterV2()
	ret.SetApproversV2(nil)
	return ret
}

// GetTestVulnRequestNoComments returns a mock *storage.VulnerabilityRequest with nil `.comments` field.
func GetTestVulnRequestNoComments(t *testing.T) *storage.VulnerabilityRequest {
	ret := GetTestVulnDeferralRequestFull(t)
	ret.SetComments(nil)
	return ret
}

// GetTestVulnRequestWithUpdate returns a mock *storage.VulnerabilityRequest with non-nil `.updateReq` field.
func GetTestVulnRequestWithUpdate(t *testing.T) *storage.VulnerabilityRequest {
	ret := GetTestVulnDeferralRequestFull(t)
	re := &storage.RequestExpiry{}
	re.SetExpiryType(storage.RequestExpiry_TIME)
	re.SetExpiresOn(proto.ValueOrDefault(ts1))
	du := &storage.DeferralUpdate{}
	du.SetExpiry(re)
	ret.SetDeferralUpdate(du)
	return ret
}
