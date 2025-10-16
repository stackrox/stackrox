package inputtypes

import (
	"github.com/graph-gophers/graphql-go"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"google.golang.org/protobuf/proto"
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
		ret.SetExpiresWhenFixed(true)
	} else if re.ExpiresOn != nil {
		ts := protoconv.ConvertTimeToTimestampOrNil(re.ExpiresOn.Time)
		if ts == nil {
			return &storage.RequestExpiry{}
		}
		ret.SetExpiresOn(proto.ValueOrDefault(ts))
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

	ret := &v1.DeferVulnRequest{}
	ret.SetCve(func() string {
		if dr.Cve == nil {
			return ""
		}
		return *dr.Cve
	}())
	ret.SetComment(func() string {
		if dr.Comment == nil {
			return ""
		}
		return *dr.Comment
	}())
	ret.SetScope(dr.Scope.AsV1VulnerabilityRequestScope())

	if dr.ExpiresWhenFixed == nil && dr.ExpiresOn == nil {
		return ret
	}
	if dr.ExpiresWhenFixed != nil {
		if *dr.ExpiresWhenFixed {
			ret.SetExpiresWhenFixed(true)
		}
	} else {
		ts := protoconv.ConvertTimeToTimestampOrNil(dr.ExpiresOn.Time)
		if ts == nil {
			return nil
		}
		ret.SetExpiresOn(proto.ValueOrDefault(ts))
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
	fpvr := &v1.FalsePositiveVulnRequest{}
	fpvr.SetCve(func() string {
		if fpr.Cve == nil {
			return ""
		}
		return *fpr.Cve
	}())
	fpvr.SetComment(func() string {
		if fpr.Comment == nil {
			return ""
		}
		return *fpr.Comment
	}())
	fpvr.SetScope(fpr.Scope.AsV1VulnerabilityRequestScope())
	return fpvr
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
		vs := &storage.VulnerabilityRequest_Scope{}
		vs.SetImageScope(proto.ValueOrDefault(rs.ImageScope.AsV1VulnerabilityRequestImageScope()))
		return vs
	}
	if rs.GlobalScope != nil {
		vs := &storage.VulnerabilityRequest_Scope{}
		vs.SetGlobalScope(proto.ValueOrDefault(rs.GlobalScope.AsV1VulnerabilityRequestGlobalScope()))
		return vs
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
	vsi := &storage.VulnerabilityRequest_Scope_Image{}
	vsi.SetRegistry(func() string {
		if rs.Registry == nil {
			return ""
		}
		return *rs.Registry
	}())
	vsi.SetRemote(func() string {
		if rs.Remote == nil {
			return ""
		}
		return *rs.Remote
	}())
	vsi.SetTag(func() string {
		if rs.Tag == nil {
			return ""
		}
		return *rs.Tag
	}())
	return vsi
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
