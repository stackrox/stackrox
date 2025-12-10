package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/views/imagecveflat"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

type imageCVEV2Resolver struct {
	ctx  context.Context
	root *Resolver
	data *storage.ImageCVEV2

	flatData imagecveflat.CveFlat
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

func (resolver *Resolver) wrapImageCVEV2FlatWithContext(ctx context.Context, value *storage.ImageCVEV2, flatData imagecveflat.CveFlat, ok bool, err error) (*imageCVEV2Resolver, error) {
	if !ok || err != nil || value == nil {
		return nil, err
	}
	return &imageCVEV2Resolver{ctx: ctx, root: resolver, data: value, flatData: flatData}, nil
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

func (resolver *Resolver) wrapImageCVEV2sFlatWithContext(ctx context.Context, values []*storage.ImageCVEV2, flatData []imagecveflat.CveFlat, err error) ([]*imageCVEV2Resolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	coreMap := make(map[string]imagecveflat.CveFlat)
	for _, res := range flatData {
		if _, ok := coreMap[res.GetCVE()]; !ok {
			coreMap[res.GetCVE()] = res
		}
	}
	output := make([]*imageCVEV2Resolver, len(values))
	for i, v := range values {
		output[i] = &imageCVEV2Resolver{ctx: ctx, root: resolver, data: v, flatData: coreMap[v.GetCveBaseInfo().GetCve()]}
	}
	return output, nil
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

func (resolver *imageCVEV2Resolver) Severity(ctx context.Context) string {
	value := resolver.data.GetSeverity()
	return value.String()
}

func (resolver *imageCVEV2Resolver) State(ctx context.Context) string {
	value := resolver.data.GetState()
	return value.String()
}
