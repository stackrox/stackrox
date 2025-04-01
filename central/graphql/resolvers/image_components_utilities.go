package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

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
	log.Infof("SHREWS -- try this.component -- FixedBy")
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
