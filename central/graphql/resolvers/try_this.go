package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/views/imagecve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

type imageCVEResolver struct {
	ctx  context.Context
	root *Resolver
	data *storage.ImageCVE

	flatData imagecve.CveCore
}

func (resolver *Resolver) wrapImageCVE(value *storage.ImageCVE, ok bool, err error) (*imageCVEResolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageCVEResolver{root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageCVEs(values []*storage.ImageCVE, err error) ([]*imageCVEResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageCVEResolver, len(values))
	for i, v := range values {
		output[i] = &imageCVEResolver{root: resolver, data: v}
	}
	return output, nil
}

func (resolver *Resolver) wrapImageCVEWithContext(ctx context.Context, value *storage.ImageCVE, ok bool, err error) (*imageCVEResolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageCVEResolver{ctx: ctx, root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageCVEsWithContext(ctx context.Context, values []*storage.ImageCVE, err error) ([]*imageCVEResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageCVEResolver, len(values))
	for i, v := range values {
		output[i] = &imageCVEResolver{ctx: ctx, root: resolver, data: v}
	}
	return output, nil
}

func (resolver *imageCVEResolver) CveBaseInfo(ctx context.Context) (*cVEInfoResolver, error) {
	value := resolver.data.GetCveBaseInfo()
	return resolver.root.wrapCVEInfo(value, true, nil)
}

func (resolver *imageCVEResolver) Cvss(ctx context.Context) float64 {
	value := resolver.data.GetCvss()
	return float64(value)
}

func (resolver *imageCVEResolver) CvssMetrics(ctx context.Context) ([]*cVSSScoreResolver, error) {
	value := resolver.data.GetCvssMetrics()
	return resolver.root.wrapCVSSScores(value, nil)
}

func (resolver *imageCVEResolver) Id(ctx context.Context) graphql.ID {
	value := resolver.data.GetId()
	return graphql.ID(value)
}

func (resolver *imageCVEResolver) ImpactScore(ctx context.Context) float64 {
	value := resolver.data.GetImpactScore()
	return float64(value)
}

func (resolver *imageCVEResolver) NvdScoreVersion(ctx context.Context) string {
	value := resolver.data.GetNvdScoreVersion()
	return value.String()
}

func (resolver *imageCVEResolver) Nvdcvss(ctx context.Context) float64 {
	value := resolver.data.GetNvdcvss()
	return float64(value)
}

func (resolver *imageCVEResolver) OperatingSystem(ctx context.Context) string {
	value := resolver.data.GetOperatingSystem()
	return value
}

func (resolver *imageCVEResolver) Severity(ctx context.Context) string {
	value := resolver.data.GetSeverity()
	return value.String()
}

func (resolver *imageCVEResolver) SnoozeExpiry(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetSnoozeExpiry()
	return protocompat.ConvertTimestampToGraphqlTimeOrError(value)
}

func (resolver *imageCVEResolver) SnoozeStart(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetSnoozeStart()
	return protocompat.ConvertTimestampToGraphqlTimeOrError(value)
}

func (resolver *imageCVEResolver) Snoozed(ctx context.Context) bool {
	value := resolver.data.GetSnoozed()
	return value
}

type imageCVEV2Resolver struct {
	ctx  context.Context
	root *Resolver
	data *storage.ImageCVEV2

	flatData imagecve.CveCore
}

func (resolver *Resolver) wrapImageCVEV2(value *storage.ImageCVEV2, ok bool, err error) (*imageCVEV2Resolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageCVEV2Resolver{root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageCVEV2s(values []*storage.ImageCVEV2, err error) ([]*imageCVEV2Resolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageCVEV2Resolver, len(values))
	for i, v := range values {
		output[i] = &imageCVEV2Resolver{root: resolver, data: v}
	}
	return output, nil
}

func (resolver *Resolver) wrapImageCVEV2WithContext(ctx context.Context, value *storage.ImageCVEV2, ok bool, err error) (*imageCVEV2Resolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageCVEV2Resolver{ctx: ctx, root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageCVEV2sWithContext(ctx context.Context, values []*storage.ImageCVEV2, err error) ([]*imageCVEV2Resolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageCVEV2Resolver, len(values))
	for i, v := range values {
		output[i] = &imageCVEV2Resolver{ctx: ctx, root: resolver, data: v}
	}
	return output, nil
}

func (resolver *Resolver) wrapImageCVEV2sCoreWithContext(ctx context.Context, values []*storage.ImageCVEV2, coreResover []*imageCVECoreResolver, err error) ([]*imageCVEV2Resolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	coreMap := make(map[string]imagecve.CveCore)
	for _, res := range coreResover {
		if _, ok := coreMap[res.CVE(ctx)]; !ok {
			coreMap[res.CVE(ctx)] = res.data
		}
	}
	output := make([]*imageCVEV2Resolver, len(values))
	for i, v := range values {
		log.Infof("SHREWS -- trying to add flat data %v", coreMap[v.GetCveBaseInfo().GetCve()])
		output[i] = &imageCVEV2Resolver{ctx: ctx, root: resolver, data: v, flatData: coreMap[v.GetCveBaseInfo().GetCve()]}
	}
	return output, nil
}

func (resolver *imageCVEV2Resolver) Advisory(ctx context.Context) string {
	value := resolver.data.GetAdvisory()
	return value
}

func (resolver *imageCVEV2Resolver) ComponentId(ctx context.Context) string {
	value := resolver.data.GetComponentId()
	return value
}

func (resolver *imageCVEV2Resolver) CveBaseInfo(ctx context.Context) (*cVEInfoResolver, error) {
	value := resolver.data.GetCveBaseInfo()
	return resolver.root.wrapCVEInfo(value, true, nil)
}

func (resolver *imageCVEV2Resolver) Cvss(ctx context.Context) float64 {
	value := resolver.data.GetCvss()
	return float64(value)
}

func (resolver *imageCVEV2Resolver) FirstImageOccurrence(ctx context.Context) (*graphql.Time, error) {
	value := resolver.data.GetFirstImageOccurrence()
	return protocompat.ConvertTimestampToGraphqlTimeOrError(value)
}

func (resolver *imageCVEV2Resolver) Id(ctx context.Context) graphql.ID {
	value := resolver.data.GetId()
	return graphql.ID(value)
}

func (resolver *imageCVEV2Resolver) ImageId(ctx context.Context) string {
	value := resolver.data.GetImageId()
	return value
}

func (resolver *imageCVEV2Resolver) ImpactScore(ctx context.Context) float64 {
	value := resolver.data.GetImpactScore()
	return float64(value)
}

func (resolver *imageCVEV2Resolver) NvdScoreVersion(ctx context.Context) string {
	value := resolver.data.GetNvdScoreVersion()
	return value.String()
}

func (resolver *imageCVEV2Resolver) Nvdcvss(ctx context.Context) float64 {
	value := resolver.data.GetNvdcvss()
	return float64(value)
}

func (resolver *imageCVEV2Resolver) OperatingSystem(ctx context.Context) string {
	value := resolver.data.GetOperatingSystem()
	return value
}

func (resolver *imageCVEV2Resolver) Severity(ctx context.Context) string {
	value := resolver.data.GetSeverity()
	return value.String()
}

func (resolver *imageCVEV2Resolver) State(ctx context.Context) string {
	value := resolver.data.GetState()
	return value.String()
}

type imageComponentResolver struct {
	ctx  context.Context
	root *Resolver
	data *storage.ImageComponent

	subFieldQuery *v1.Query
}

func (resolver *Resolver) wrapImageComponent(value *storage.ImageComponent, ok bool, err error) (*imageComponentResolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageComponentResolver{root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageComponents(values []*storage.ImageComponent, err error) ([]*imageComponentResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageComponentResolver, len(values))
	for i, v := range values {
		output[i] = &imageComponentResolver{root: resolver, data: v}
	}
	return output, nil
}

func (resolver *Resolver) wrapImageComponentWithContext(ctx context.Context, value *storage.ImageComponent, ok bool, err error) (*imageComponentResolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageComponentResolver{ctx: ctx, root: resolver, data: value}, nil
}

func (resolver *Resolver) wrapImageComponentsWithContext(ctx context.Context, values []*storage.ImageComponent, err error) ([]*imageComponentResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageComponentResolver, len(values))
	for i, v := range values {
		output[i] = &imageComponentResolver{ctx: ctx, root: resolver, data: v}
	}
	return output, nil
}

func (resolver *imageComponentResolver) FixedBy(ctx context.Context) string {
	value := resolver.data.GetFixedBy()
	return value
}

func (resolver *imageComponentResolver) Id(ctx context.Context) graphql.ID {
	value := resolver.data.GetId()
	return graphql.ID(value)
}

func (resolver *imageComponentResolver) License(ctx context.Context) (*licenseResolver, error) {
	value := resolver.data.GetLicense()
	return resolver.root.wrapLicense(value, true, nil)
}

func (resolver *imageComponentResolver) Name(ctx context.Context) string {
	value := resolver.data.GetName()
	return value
}

func (resolver *imageComponentResolver) OperatingSystem(ctx context.Context) string {
	value := resolver.data.GetOperatingSystem()
	return value
}

func (resolver *imageComponentResolver) Priority(ctx context.Context) int32 {
	value := resolver.data.GetPriority()
	return int32(value)
}

func (resolver *imageComponentResolver) RiskScore(ctx context.Context) float64 {
	value := resolver.data.GetRiskScore()
	return float64(value)
}

func (resolver *imageComponentResolver) Source(ctx context.Context) string {
	value := resolver.data.GetSource()
	return value.String()
}

func (resolver *imageComponentResolver) Version(ctx context.Context) string {
	value := resolver.data.GetVersion()
	return value
}

type imageComponentV2Resolver struct {
	ctx  context.Context
	root *Resolver
	data *storage.ImageComponentV2

	subFieldQuery PaginatedQuery
}

func (resolver *Resolver) wrapImageComponentV2(value *storage.ImageComponentV2, ok bool, subFieldQuery PaginatedQuery, err error) (*imageComponentV2Resolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageComponentV2Resolver{root: resolver, data: value, subFieldQuery: subFieldQuery}, nil
}

func (resolver *Resolver) wrapImageComponentV2s(values []*storage.ImageComponentV2, subFieldQuery PaginatedQuery, err error) ([]*imageComponentV2Resolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageComponentV2Resolver, len(values))
	for i, v := range values {
		output[i] = &imageComponentV2Resolver{root: resolver, data: v, subFieldQuery: subFieldQuery}
	}
	return output, nil
}

func (resolver *Resolver) wrapImageComponentV2WithContext(ctx context.Context, value *storage.ImageComponentV2, ok bool, subFieldQuery PaginatedQuery, err error) (*imageComponentV2Resolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageComponentV2Resolver{ctx: ctx, root: resolver, data: value, subFieldQuery: subFieldQuery}, nil
}

func (resolver *Resolver) wrapImageComponentV2sWithContext(ctx context.Context, values []*storage.ImageComponentV2, subFieldQuery PaginatedQuery, err error) ([]*imageComponentV2Resolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageComponentV2Resolver, len(values))
	for i, v := range values {
		output[i] = &imageComponentV2Resolver{ctx: ctx, root: resolver, data: v, subFieldQuery: subFieldQuery}
	}
	return output, nil
}

func (resolver *imageComponentV2Resolver) Architecture(ctx context.Context) string {
	value := resolver.data.GetArchitecture()
	return value
}

func (resolver *imageComponentV2Resolver) FixedBy(ctx context.Context) string {
	value := resolver.data.GetFixedBy()
	return value
}

func (resolver *imageComponentV2Resolver) Id(ctx context.Context) graphql.ID {
	value := resolver.data.GetId()
	return graphql.ID(value)
}

func (resolver *imageComponentV2Resolver) ImageId(ctx context.Context) string {
	value := resolver.data.GetImageId()
	return value
}

func (resolver *imageComponentV2Resolver) License(ctx context.Context) (*licenseResolver, error) {
	value := resolver.data.GetLicense()
	return resolver.root.wrapLicense(value, true, nil)
}

func (resolver *imageComponentV2Resolver) Name(ctx context.Context) string {
	value := resolver.data.GetName()
	return value
}

func (resolver *imageComponentV2Resolver) OperatingSystem(ctx context.Context) string {
	value := resolver.data.GetOperatingSystem()
	return value
}

func (resolver *imageComponentV2Resolver) Priority(ctx context.Context) int32 {
	value := resolver.data.GetPriority()
	return int32(value)
}

func (resolver *imageComponentV2Resolver) RiskScore(ctx context.Context) float64 {
	value := resolver.data.GetRiskScore()
	return float64(value)
}

func (resolver *imageComponentV2Resolver) Source(ctx context.Context) string {
	value := resolver.data.GetSource()
	return value.String()
}

func (resolver *imageComponentV2Resolver) Version(ctx context.Context) string {
	value := resolver.data.GetVersion()
	return value
}

func toCvssScoreVersion(value *string) storage.CvssScoreVersion {
	if value != nil {
		return storage.CvssScoreVersion(storage.CvssScoreVersion_value[*value])
	}
	return storage.CvssScoreVersion(0)
}

func toCvssScoreVersions(values *[]string) []storage.CvssScoreVersion {
	if values == nil {
		return nil
	}
	output := make([]storage.CvssScoreVersion, len(*values))
	for i, v := range *values {
		output[i] = toCvssScoreVersion(&v)
	}
	return output
}
