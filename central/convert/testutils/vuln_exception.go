package testutils

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/proto"
)

var (
	ts1 = protocompat.TimestampNow()
)

// GetTestVulnDeferralExceptionFull returns a mock *v2.VulnerabilityException of deferral kind.
func GetTestVulnDeferralExceptionFull(_ *testing.T) *v2.VulnerabilityException {
	return v2.VulnerabilityException_builder{
		Id:          "id",
		Name:        "name",
		TargetState: v2.VulnerabilityState_DEFERRED,
		Status:      v2.ExceptionStatus_PENDING,
		Expired:     false,
		Requester: v2.SlimUser_builder{
			Id:   "userID",
			Name: "userName",
		}.Build(),
		Approvers: []*v2.SlimUser{
			v2.SlimUser_builder{
				Id:   "userID",
				Name: "userName",
			}.Build(),
		},
		Comments: []*v2.Comment{
			v2.Comment_builder{
				Id:      "commentID",
				Message: "message",
				User: v2.SlimUser_builder{
					Id:   "userID",
					Name: "userName",
				}.Build(),
			}.Build(),
		},
		DeferralRequest: v2.DeferralRequest_builder{
			Expiry: v2.ExceptionExpiry_builder{
				ExpiryType: v2.ExceptionExpiry_TIME,
				ExpiresOn:  ts1,
			}.Build(),
		}.Build(),
		Scope: v2.VulnerabilityException_Scope_builder{
			ImageScope: v2.VulnerabilityException_Scope_Image_builder{
				Registry: "reg",
				Remote:   "remote",
				Tag:      "tag",
			}.Build(),
		}.Build(),
		Cves: []string{"cve1"},
	}.Build()
}

// GetTestVulnFPExceptionFull returns a mock *v2.VulnerabilityException of false-positive kind.
func GetTestVulnFPExceptionFull(t *testing.T) *v2.VulnerabilityException {
	ret := GetTestVulnDeferralExceptionFull(t)
	ret.SetTargetState(v2.VulnerabilityState_FALSE_POSITIVE)
	ret.SetFalsePositiveRequest(&v2.FalsePositiveRequest{})
	return ret
}

// GetTestVulnExceptionNoUsers returns a mock *v2.VulnerabilityException with nil `.requester` and `.approvers` fields.
func GetTestVulnExceptionNoUsers(t *testing.T) *v2.VulnerabilityException {
	ret := GetTestVulnDeferralExceptionFull(t)
	ret.ClearRequester()
	ret.SetApprovers(nil)
	return ret
}

// GetTestVulnExceptionNoComments returns a mock *v2.VulnerabilityException with nil `.comments` field.
func GetTestVulnExceptionNoComments(t *testing.T) *v2.VulnerabilityException {
	ret := GetTestVulnDeferralExceptionFull(t)
	ret.SetComments(nil)
	return ret
}

// GetTestVulnExceptionWithUpdate returns a mock *v2.VulnerabilityException with non-nil `.updateReq` field.
func GetTestVulnExceptionWithUpdate(t *testing.T) *v2.VulnerabilityException {
	ret := GetTestVulnDeferralExceptionFull(t)
	ee := &v2.ExceptionExpiry{}
	ee.SetExpiryType(v2.ExceptionExpiry_TIME)
	ee.SetExpiresOn(ts1)
	du := &v2.DeferralUpdate{}
	du.SetExpiry(ee)
	ret.SetDeferralUpdate(proto.ValueOrDefault(du))
	return ret
}

// GetTestCreateDeferVulnExceptionRequest returns a mock *v2.CreateDeferVulnerabilityExceptionRequest.
func GetTestCreateDeferVulnExceptionRequest(t *testing.T) *v2.CreateDeferVulnerabilityExceptionRequest {
	req := GetTestVulnDeferralExceptionFull(t)
	cdver := &v2.CreateDeferVulnerabilityExceptionRequest{}
	cdver.SetCves([]string{"cve1"})
	cdver.SetComment("message")
	cdver.SetScope(req.GetScope())
	cdver.SetExceptionExpiry(req.GetDeferralRequest().GetExpiry())
	return cdver
}

// GetTestCreateFPVulnExceptionRequest returns a mock *v2.CreateFalsePositiveVulnerabilityExceptionRequest.
func GetTestCreateFPVulnExceptionRequest(t *testing.T) *v2.CreateFalsePositiveVulnerabilityExceptionRequest {
	cfpver := &v2.CreateFalsePositiveVulnerabilityExceptionRequest{}
	cfpver.SetCves([]string{"cve1"})
	cfpver.SetComment("message")
	cfpver.SetScope(GetTestVulnDeferralExceptionFull(t).GetScope())
	return cfpver
}
