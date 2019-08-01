package search

import (
	"context"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// We need to handle the fact that the Image.Name.FullName field uses a different analyzer. To do this, when we see a
// sort on this field, we swap it for a compound sort on the other fields of the Image.Name.
////////////////////////////////////////////////////////////////////////////////////////////
func swapImageNameSortOption(searcher search.Searcher) search.Searcher {
	return &swapImageNameSortOptionImpl{
		searcher: searcher,
	}
}

type swapImageNameSortOptionImpl struct {
	searcher search.Searcher
}

func (ds *swapImageNameSortOptionImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	if len(q.GetPagination().GetSortOptions()) == 0 {
		return ds.searcher.Search(ctx, q)
	}

	newSortFields := make([]*v1.QuerySortOption, 0, len(q.GetPagination().GetSortOptions()))
	for _, so := range q.GetPagination().GetSortOptions() {
		if so.GetField() == search.ImageName.String() {
			newSortFields = append(newSortFields, swapImageName(so)...)
		} else {
			newSortFields = append(newSortFields, so)
		}
	}

	newQuery := proto.Clone(q).(*v1.Query)
	newQuery.Pagination.SortOptions = newSortFields
	return ds.searcher.Search(ctx, newQuery)
}

func swapImageName(in *v1.QuerySortOption) []*v1.QuerySortOption {
	return []*v1.QuerySortOption{
		{
			Field:    search.ImageRegistry.String(),
			Reversed: in.Reversed,
		},
		{
			Field:    search.ImageRemote.String(),
			Reversed: in.Reversed,
		},
		{
			Field:    search.ImageTag.String(),
			Reversed: in.Reversed,
		},
	}
}
