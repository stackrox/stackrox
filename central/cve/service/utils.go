package service

//
// func suppressCVEReqToVulnReq(request *v1.SuppressCVERequest, createdAt *types.Timestamp) *storage.VulnerabilityRequest {
//	d, err := types.DurationFromProto(request.GetDuration())
//	if err != nil {
//		log.Errorf("could not create vulnerability request for CVE(s) %v", request.GetIds())
//		return nil
//	}
//
//	return &storage.VulnerabilityRequest{
//		Expired:     false,
//		TargetState: storage.VulnerabilityState_DEFERRED,
//		Status:      storage.RequestStatus_APPROVED,
//		Scope: &storage.VulnerabilityRequest_Scope{
//			Info: &storage.VulnerabilityRequest_Scope_GlobalScope{
//				GlobalScope: &storage.VulnerabilityRequest_Scope_Global{},
//			},
//		},
//		Entities: &storage.VulnerabilityRequest_Cves{
//			Cves: &storage.VulnerabilityRequest_CVEs{
//				Ids: request.GetIds(),
//			},
//		},
//		Req: &storage.VulnerabilityRequest_DeferralReq{
//			DeferralReq: &storage.DeferralRequest{
//				Expiry: &storage.RequestExpiry{
//					Expiry: &storage.RequestExpiry_ExpiresOn{
//						ExpiresOn: &types.Timestamp{Seconds: createdAt.GetSeconds() + int64(d.Seconds())},
//					},
//				},
//			},
//		},
//	}
// }
//
// func unSuppressCVEReqToVulnReq(request *v1.UnsuppressCVERequest) *storage.VulnerabilityRequest {
//	return &storage.VulnerabilityRequest{
//		TargetState: storage.VulnerabilityState_DEFERRED,
//		Status:      storage.RequestStatus_APPROVED,
//		Scope: &storage.VulnerabilityRequest_Scope{
//			Info: &storage.VulnerabilityRequest_Scope_GlobalScope{
//				GlobalScope: &storage.VulnerabilityRequest_Scope_Global{},
//			},
//		},
//		Entities: &storage.VulnerabilityRequest_Cves{
//			Cves: &storage.VulnerabilityRequest_CVEs{
//				Ids: request.GetIds(),
//			},
//		},
//	}
// }
