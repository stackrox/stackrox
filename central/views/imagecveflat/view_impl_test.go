package imagecveflat

import (
	"testing"

	"github.com/stackrox/rox/central/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSelectCVENamesQuery(t *testing.T) {
	base := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(20)).ProtoQuery()

	q := withSelectCVENamesQuery(base)

	require.Len(t, q.GetSelects(), 1)
	assert.Equal(t, search.CVE.String(), q.GetSelects()[0].GetField().GetName())
	assert.False(t, q.GetSelects()[0].GetField().GetDistinct())
	assert.Equal(t, []string{search.CVE.String()}, q.GetGroupBy().GetFields())
}

func TestWithSelectCVEFlatResponseQueryFiltering(t *testing.T) {
	cvesToFilter := []string{"CVE-2024-0001", "CVE-2024-0002"}

	assertFilterMode := func(t *testing.T, q *v1.Query, wantCVEName bool) {
		t.Helper()
		var sawCVEName, sawDocIDs bool
		search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
			if mfq := bq.GetMatchFieldQuery(); mfq != nil && mfq.GetField() == search.CVE.String() {
				sawCVEName = true
			}
			if bq.GetDocIdQuery() != nil {
				sawDocIDs = true
			}
		})
		if wantCVEName {
			assert.True(t, sawCVEName, "expected a CVE name filter")
			assert.False(t, sawDocIDs, "did not expect a row ID filter")
			return
		}
		assert.False(t, sawCVEName, "did not expect a CVE name filter")
		assert.True(t, sawDocIDs, "expected a row ID filter")
	}

	t.Run("filter by CVE name", func(t *testing.T) {
		base := search.NewQueryBuilder().ProtoQuery()
		q := withSelectCVEFlatResponseQuery(base, cvesToFilter, views.ReadOptions{UseCVENameFiltering: true})
		assertFilterMode(t, q, true)
	})

	t.Run("filter by row ID", func(t *testing.T) {
		base := search.NewQueryBuilder().ProtoQuery()
		q := withSelectCVEFlatResponseQuery(base, cvesToFilter, views.ReadOptions{})
		assertFilterMode(t, q, false)
	})
}
