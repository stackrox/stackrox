package images

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func sortOpt(field string, reversed bool) *v1.QuerySortOption {
	return &v1.QuerySortOption{Field: field, Reversed: reversed}
}

func TestExpandSeverityRankSort(t *testing.T) {
	storedDesc := []*v1.QuerySortOption{
		sortOpt(search.CriticalCVECount.String(), true),
		sortOpt(search.ImportantCVECount.String(), true),
		sortOpt(search.ModerateCVECount.String(), true),
		sortOpt(search.LowCVECount.String(), true),
		sortOpt(search.UnknownCVECount.String(), true),
	}

	cases := map[string]struct {
		in       []*v1.QuerySortOption
		expected []*v1.QuerySortOption
	}{
		"no sort options is a no-op": {
			in:       nil,
			expected: nil,
		},
		"unrelated sort is untouched": {
			in:       []*v1.QuerySortOption{sortOpt(search.ImageName.String(), false)},
			expected: []*v1.QuerySortOption{sortOpt(search.ImageName.String(), false)},
		},
		"severity rank expands to stored counts (critical first) plus unique tiebreaker": {
			in: []*v1.QuerySortOption{sortOpt(SeverityRankSortField, true)},
			expected: append(append([]*v1.QuerySortOption{}, storedDesc...),
				sortOpt(search.ImageID.String(), false)),
		},
		"direction is propagated to all expanded fields": {
			in: []*v1.QuerySortOption{sortOpt(SeverityRankSortField, false)},
			expected: []*v1.QuerySortOption{
				sortOpt(search.CriticalCVECount.String(), false),
				sortOpt(search.ImportantCVECount.String(), false),
				sortOpt(search.ModerateCVECount.String(), false),
				sortOpt(search.LowCVECount.String(), false),
				sortOpt(search.UnknownCVECount.String(), false),
				sortOpt(search.ImageID.String(), false),
			},
		},
		"surrounding sort options are preserved in order": {
			in: []*v1.QuerySortOption{
				sortOpt(search.ImageName.String(), false),
				sortOpt(SeverityRankSortField, true),
				sortOpt(search.ImageOS.String(), false),
			},
			expected: append(
				append([]*v1.QuerySortOption{sortOpt(search.ImageName.String(), false)}, storedDesc...),
				sortOpt(search.ImageID.String(), false),
				sortOpt(search.ImageOS.String(), false),
			),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			q := &v1.Query{}
			if tc.in != nil {
				q.Pagination = &v1.QueryPagination{SortOptions: tc.in}
			}

			expandSeverityRankSort(q, search.ImageID)
			assert.Equal(t, tc.expected, q.GetPagination().GetSortOptions())
		})
	}
}
