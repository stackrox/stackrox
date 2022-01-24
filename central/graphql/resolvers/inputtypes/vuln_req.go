package inputtypes

import (
	"github.com/graph-gophers/graphql-go"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

// VulnReqExpiry represents when a vulnerability request can expire.
type VulnReqExpiry struct {
	ExpiresWhenFixed *bool
	ExpiresOn        *graphql.Time
}

// AsRequestExpiry converts vulnerability request expiry to proto.
func (re *VulnReqExpiry) AsRequestExpiry() *storage.RequestExpiry {
	if re == nil {
		return &storage.RequestExpiry{}
	}

	ret := &storage.RequestExpiry{}
	if re.ExpiresWhenFixed != nil && *re.ExpiresWhenFixed {
		ret.Expiry = &storage.RequestExpiry_ExpiresWhenFixed{
			ExpiresWhenFixed: true,
		}
	} else if re.ExpiresOn != nil {
		ts := protoconv.ConvertTimeToTimestampOrNil(re.ExpiresOn.Time)
		if ts == nil {
			return &storage.RequestExpiry{}
		}
		ret.Expiry = &storage.RequestExpiry_ExpiresOn{
			ExpiresOn: ts,
		}
	}
	return ret
}

// DeferVulnRequest encapsulates the request data for vulnerability deferral request.
type DeferVulnRequest struct {
	Cve              *string
	Comment          *string
	Scope            *VulnReqScope
	ExpiresWhenFixed *bool
	ExpiresOn        *graphql.Time
}

// AsV1DeferralRequest converts the deferral request option to proto.
func (dr *DeferVulnRequest) AsV1DeferralRequest() *v1.DeferVulnRequest {
	if dr == nil {
		return nil
	}

	ret := &v1.DeferVulnRequest{
		Cve: func() string {
			if dr.Cve == nil {
				return ""
			}
			return *dr.Cve
		}(),
		Comment: func() string {
			if dr.Comment == nil {
				return ""
			}
			return *dr.Comment
		}(),
		Scope: dr.Scope.AsV1VulnerabilityRequestScope(),
	}

	if dr.ExpiresWhenFixed == nil && dr.ExpiresOn == nil {
		return ret
	}
	if dr.ExpiresWhenFixed != nil {
		if *dr.ExpiresWhenFixed {
			ret.Expiry = &v1.DeferVulnRequest_ExpiresWhenFixed{
				ExpiresWhenFixed: true,
			}
		}
	} else {
		ts := protoconv.ConvertTimeToTimestampOrNil(dr.ExpiresOn.Time)
		if ts == nil {
			return nil
		}
		ret.Expiry = &v1.DeferVulnRequest_ExpiresOn{
			ExpiresOn: ts,
		}
	}
	return ret
}

// FalsePositiveVulnRequest encapsulates the request data to mark the vulnerability as false-positive.
type FalsePositiveVulnRequest struct {
	Cve     *string
	Comment *string
	Scope   *VulnReqScope
}

// AsV1FalsePositiveRequest converts the false positive request option to proto.
func (fpr *FalsePositiveVulnRequest) AsV1FalsePositiveRequest() *v1.FalsePositiveVulnRequest {
	if fpr == nil {
		return nil
	}
	return &v1.FalsePositiveVulnRequest{
		Cve: func() string {
			if fpr.Cve == nil {
				return ""
			}
			return *fpr.Cve
		}(),
		Comment: func() string {
			if fpr.Comment == nil {
				return ""
			}
			return *fpr.Comment
		}(),
		Scope: fpr.Scope.AsV1VulnerabilityRequestScope(),
	}
}

// VulnReqScope represents the scope of vulnerability request.
type VulnReqScope struct {
	ImageScope  *VulnReqImageScope
	GlobalScope *VulnReqGlobalScope
}

// AsV1VulnerabilityRequestScope converts vulnerability request scope to proto.
func (rs *VulnReqScope) AsV1VulnerabilityRequestScope() *storage.VulnerabilityRequest_Scope {
	if rs == nil {
		return nil
	}
	if rs.ImageScope != nil {
		return &storage.VulnerabilityRequest_Scope{
			Info: &storage.VulnerabilityRequest_Scope_ImageScope{
				ImageScope: rs.ImageScope.AsV1VulnerabilityRequestImageScope(),
			},
		}
	}
	if rs.GlobalScope != nil {
		return &storage.VulnerabilityRequest_Scope{
			Info: &storage.VulnerabilityRequest_Scope_GlobalScope{
				GlobalScope: rs.GlobalScope.AsV1VulnerabilityRequestGlobalScope(),
			},
		}
	}
	return nil
}

// VulnReqImageScope represents the image scope of a vulnerability request.
type VulnReqImageScope struct {
	Registry *string
	Remote   *string
	Tag      *string
}

// AsV1VulnerabilityRequestImageScope converts vulnerability request image scope to proto.
func (rs *VulnReqImageScope) AsV1VulnerabilityRequestImageScope() *storage.VulnerabilityRequest_Scope_Image {
	if rs == nil {
		return nil
	}
	return &storage.VulnerabilityRequest_Scope_Image{
		Registry: func() string {
			if rs.Registry == nil {
				return ""
			}
			return *rs.Registry
		}(),
		Remote: func() string {
			if rs.Remote == nil {
				return ""
			}
			return *rs.Remote
		}(),
		Tag: func() string {
			if rs.Tag == nil {
				return ""
			}
			return *rs.Tag
		}(),
	}
}

// VulnReqGlobalScope represents the global scope of a vulnerability request.
type VulnReqGlobalScope struct {
	Images *VulnReqImageScope
}

// AsV1VulnerabilityRequestGlobalScope converts vulnerability request global scope to proto.
func (rs *VulnReqGlobalScope) AsV1VulnerabilityRequestGlobalScope() *storage.VulnerabilityRequest_Scope_Global {
	if rs == nil || rs.Images == nil {
		return nil
	}
	if *rs.Images.Registry != ".*" || *rs.Images.Remote != ".*" || *rs.Images.Tag != ".*" {
		return nil
	}
	return &storage.VulnerabilityRequest_Scope_Global{}
}
