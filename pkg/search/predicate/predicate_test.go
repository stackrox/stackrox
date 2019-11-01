package predicate

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestSearchPredicate(t *testing.T) {
	imageFactory := NewFactory(&storage.Image{})

	baseTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	assert.NoError(t, err)

	// Pass the predicate
	ts, err := types.TimestampProto(baseTime.Add(time.Hour))
	assert.NoError(t, err)
	passingImage := &storage.Image{
		Id: "sha",
		SetCves: &storage.Image_Cves{
			Cves: 3,
		},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "firstComponent",
					Version: "1.0",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: "cve-2018-1",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.1",
							},
						},
					},
				},
				{
					Name:    "SecondComponent",
					Version: "1.1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: "cve-2018-1",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.5",
							},
						},
					},
				},
				{
					Name:    "ThirdComponent",
					Version: "1.0",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: "cve-2019-1",
						},
						{
							Cve: "cve-2019-2",
						},
					},
				},
			},
		},
		LastUpdated: ts,
	}

	cases := []struct {
		name        string
		query       *v1.Query
		expectation bool
	}{
		{
			name:        "empty query",
			query:       &v1.Query{},
			expectation: true,
		},
		{
			name: "basic conjunction",
			query: search.NewQueryBuilder().
				AddStrings(search.ImageSHA, "sha").
				AddStrings(search.CVECount, "<4").
				AddStrings(search.LastUpdatedTime, ">2006-01-02T15:04:05Z").
				AddStrings(search.FixedBy, "1.1").
				ProtoQuery(),
			expectation: true,
		},
		{
			name: "linked fields within struct match",
			query: search.NewQueryBuilder().
				AddLinkedFields(
					[]search.FieldLabel{search.CVE, search.FixedBy},
					[]string{search.ExactMatchString("cve-2018-1"), search.RegexQueryString(".+")},
				).
				ProtoQuery(),
			expectation: true,
		},
		{
			name: "linked fields within struct do not match",
			query: search.NewQueryBuilder().
				AddLinkedFields(
					[]search.FieldLabel{search.CVE, search.FixedBy},
					[]string{search.ExactMatchString("cve-2019-1"), search.RegexQueryString(".+")},
				).
				ProtoQuery(),
			expectation: false,
		},
		{
			name: "nested linked fields within struct match",
			query: search.NewQueryBuilder().
				AddLinkedFields(
					[]search.FieldLabel{search.Component, search.CVE},
					[]string{search.ExactMatchString("ThirdComponent"), search.ExactMatchString("cve-2019-1")},
				).
				ProtoQuery(),
			expectation: true,
		},
		{
			name: "nested linked fields within struct do not match",
			query: search.NewQueryBuilder().
				AddLinkedFields(
					[]search.FieldLabel{search.Component, search.CVE},
					[]string{search.ExactMatchString("ThirdComponent"), search.ExactMatchString("cve-2018-1")},
				).
				ProtoQuery(),
			expectation: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pred, err := imageFactory.GeneratePredicate(c.query)
			assert.NotNil(t, pred)
			assert.NoError(t, err)
			assert.Equal(t, c.expectation, pred(passingImage))
		})
	}
}
