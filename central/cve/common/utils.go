package common

import (
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/proto"
)

var (
	log = logging.LoggerForModule()
)

// SuppressCVEReqToVulnReq builds a `storage.VulnerabilityRequest` (added in v2 CVE deferral workflow) from `v1.SuppressCVERequest` (legacy CVE deferral workflow).
func SuppressCVEReqToVulnReq(request *v1.SuppressCVERequest, createdAt time.Time) *storage.VulnerabilityRequest {
	d, err := protocompat.DurationFromProto(request.GetDuration())
	if err != nil {
		log.Errorf("could not create vulnerability request for CVE(s) %v", request.GetCves())
		return nil
	}
	expiresOn := createdAt.Add(d).Truncate(time.Second)

	return storage.VulnerabilityRequest_builder{
		Expired:     false,
		TargetState: storage.VulnerabilityState_DEFERRED,
		Status:      storage.RequestStatus_APPROVED,
		Scope: storage.VulnerabilityRequest_Scope_builder{
			GlobalScope: &storage.VulnerabilityRequest_Scope_Global{},
		}.Build(),
		Cves: storage.VulnerabilityRequest_CVEs_builder{
			Cves: request.GetCves(),
		}.Build(),
		DeferralReq: storage.DeferralRequest_builder{
			Expiry: storage.RequestExpiry_builder{
				ExpiresOn: proto.ValueOrDefault(protocompat.ConvertTimeToTimestampOrNil(&expiresOn)),
			}.Build(),
		}.Build(),
	}.Build()
}

// UnSuppressCVEReqToVulnReq builds a `storage.VulnerabilityRequest` (added in v2 CVE deferral workflow) from `v1.UnsuppressCVERequest` (legacy CVE deferral workflow).
func UnSuppressCVEReqToVulnReq(request *v1.UnsuppressCVERequest) *storage.VulnerabilityRequest {
	vc := &storage.VulnerabilityRequest_CVEs{}
	vc.SetCves(request.GetCves())
	vr := &storage.VulnerabilityRequest{}
	vr.SetTargetState(storage.VulnerabilityState_DEFERRED)
	vr.SetStatus(storage.RequestStatus_APPROVED)
	vr.SetScope(storage.VulnerabilityRequest_Scope_builder{
		GlobalScope: &storage.VulnerabilityRequest_Scope_Global{},
	}.Build())
	vr.SetCves(proto.ValueOrDefault(vc))
	return vr
}
