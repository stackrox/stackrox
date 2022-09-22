package common

import (
	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// SuppressCVEReqToVulnReq builds a `storage.VulnerabilityRequest` (added in v2 CVE deferral workflow) from `v1.SuppressCVERequest` (legacy CVE deferral workflow).
func SuppressCVEReqToVulnReq(request *v1.SuppressCVERequest, createdAt *types.Timestamp) *storage.VulnerabilityRequest {
	d, err := types.DurationFromProto(request.GetDuration())
	if err != nil {
		log.Errorf("could not create vulnerability request for CVE(s) %v", request.GetCves())
		return nil
	}

	return &storage.VulnerabilityRequest{
		Expired:     false,
		TargetState: storage.VulnerabilityState_DEFERRED,
		Status:      storage.RequestStatus_APPROVED,
		Scope: &storage.VulnerabilityRequest_Scope{
			Info: &storage.VulnerabilityRequest_Scope_GlobalScope{
				GlobalScope: &storage.VulnerabilityRequest_Scope_Global{},
			},
		},
		Entities: &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Ids: request.GetCves(),
			},
		},
		Req: &storage.VulnerabilityRequest_DeferralReq{
			DeferralReq: &storage.DeferralRequest{
				Expiry: &storage.RequestExpiry{
					Expiry: &storage.RequestExpiry_ExpiresOn{
						ExpiresOn: &types.Timestamp{Seconds: createdAt.GetSeconds() + int64(d.Seconds())},
					},
				},
			},
		},
	}
}

// UnSuppressCVEReqToVulnReq builds a `storage.VulnerabilityRequest` (added in v2 CVE deferral workflow) from `v1.UnsuppressCVERequest` (legacy CVE deferral workflow).
func UnSuppressCVEReqToVulnReq(request *v1.UnsuppressCVERequest) *storage.VulnerabilityRequest {
	return &storage.VulnerabilityRequest{
		TargetState: storage.VulnerabilityState_DEFERRED,
		Status:      storage.RequestStatus_APPROVED,
		Scope: &storage.VulnerabilityRequest_Scope{
			Info: &storage.VulnerabilityRequest_Scope_GlobalScope{
				GlobalScope: &storage.VulnerabilityRequest_Scope_Global{},
			},
		},
		Entities: &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Ids: request.GetCves(),
			},
		},
	}
}
