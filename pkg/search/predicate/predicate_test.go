package predicate

import (
	"fmt"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestTimePredicate(t *testing.T) {
	imageFactory := NewFactory("image", &storage.Image{})

	cases := []struct {
		imageQueryString string
		imageScanDate    time.Time
		expectedMatch    bool
	}{
		{
			imageQueryString: ">30d",
			imageScanDate:    time.Now().Add(-31 * 24 * time.Hour),
			expectedMatch:    true,
		},
		{
			imageQueryString: "<30d",
			imageScanDate:    time.Now().Add(-31 * 24 * time.Hour),
			expectedMatch:    false,
		},
		{
			imageQueryString: "<30d",
			imageScanDate:    time.Now().Add(-10 * 24 * time.Hour),
			expectedMatch:    true,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s-%s", c.imageQueryString, c.imageScanDate.String()), func(t *testing.T) {
			img := fixtures.GetImage()
			imageScan := &storage.ImageScan{}
			imageScan.SetScanTime(protoconv.ConvertTimeToTimestamp(c.imageScanDate))
			img.SetScan(imageScan)
			predicate, err := imageFactory.GeneratePredicate(search.NewQueryBuilder().AddStringsHighlighted(search.ImageScanTime, c.imageQueryString).ProtoQuery())
			require.NoError(t, err)
			assert.Equal(t, c.expectedMatch, predicate.Matches(img))
		})
	}
}

func TestSearchPredicate(t *testing.T) {
	imageFactory := NewFactory("image", &storage.Image{})
	deploymentFactory := NewFactory("deployment", &storage.Deployment{})

	baseTime, err := time.Parse(time.RFC3339, "2011-01-02T15:04:05Z")
	assert.NoError(t, err)

	// Pass the predicate
	ts, err := protocompat.ConvertTimeToTimestampOrError(baseTime.Add(time.Hour))
	assert.NoError(t, err)
	passingImage := storage.Image_builder{
		Id: "sha",
		Name: storage.ImageName_builder{
			FullName: "averygoodname",
		}.Build(),
		Cves: proto.Int32(3),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Labels: map[string]string{
					"labelOne": "test.label.one",
					"labelTwo": "test.label.two",
				},
			}.Build(),
		}.Build(),
		Scan: storage.ImageScan_builder{
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{
					Name:    "firstComponent",
					Version: "1.0",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:          "cve-2018-1",
							FixedBy:      proto.String("1.1"),
							ScoreVersion: storage.EmbeddedVulnerability_V2,
						}.Build(),
					},
				}.Build(),
				storage.EmbeddedImageScanComponent_builder{
					Name:    "SecondComponent",
					Version: "1.1",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:          "cve-2018-1",
							FixedBy:      proto.String("1.5"),
							ScoreVersion: storage.EmbeddedVulnerability_V3,
						}.Build(),
					},
				}.Build(),
				storage.EmbeddedImageScanComponent_builder{
					Name:    "ThirdComponent",
					Version: "1.0",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:          "cve-2019-1",
							ScoreVersion: storage.EmbeddedVulnerability_V3,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:          "cve-2019-2",
							ScoreVersion: storage.EmbeddedVulnerability_V2,
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
		LastUpdated: ts,
	}.Build()

	deployment := &storage.Deployment{}
	deployment.SetName("foo")
	deployment.SetNamespace("bar")

	cases := []struct {
		name        string
		query       *v1.Query
		factory     Factory
		object      interface{}
		expectation bool
	}{
		{
			name:        "empty query",
			query:       &v1.Query{},
			factory:     imageFactory,
			object:      passingImage,
			expectation: true,
		},
		{
			name: "basic conjunction",
			query: search.NewQueryBuilder().
				AddStrings(search.ImageSHA, "sha").
				AddStrings(search.CVECount, "<4").
				AddStrings(search.LastUpdatedTime, ">03/04/2010 PST").
				AddStrings(search.FixedBy, "1.1").
				ProtoQuery(),
			factory:     imageFactory,
			object:      passingImage,
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
			factory:     imageFactory,
			object:      passingImage,
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
			factory:     imageFactory,
			object:      passingImage,
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
			factory:     imageFactory,
			object:      passingImage,
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
			factory:     imageFactory,
			object:      passingImage,
			expectation: false,
		},
		{
			name: "linked fields at top level within struct match",
			query: search.NewQueryBuilder().
				AddLinkedFields(
					[]search.FieldLabel{search.DeploymentName, search.Namespace},
					[]string{search.ExactMatchString("foo"), search.ExactMatchString("bar")},
				).
				ProtoQuery(),
			factory:     deploymentFactory,
			object:      deployment,
			expectation: true,
		},
		{
			name: "linked fields at top level within struct do not match",
			query: search.NewQueryBuilder().
				AddLinkedFields(
					[]search.FieldLabel{search.DeploymentName, search.Namespace},
					[]string{search.ExactMatchString("foo"), search.ExactMatchString("foo")},
				).
				ProtoQuery(),
			factory:     deploymentFactory,
			object:      deployment,
			expectation: false,
		},
		{
			name: "negated exact match matches different strings",
			query: v1.Query_builder{
				BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{
						Field:     "Image",
						Value:     "!\"abcd\"",
						Highlight: false,
					}.Build(),
				}.Build(),
			}.Build(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: true,
		},
		{
			name: "negated exact match does not match the same string",
			query: v1.Query_builder{
				BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{
						Field:     "Image",
						Value:     "!\"averygoodname\"",
						Highlight: false,
					}.Build(),
				}.Build(),
			}.Build(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: false,
		},
		{
			name: "negated prefix query does not match a string with a matching prefix",
			query: v1.Query_builder{
				BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{
						Field:     "Image",
						Value:     "!averygood",
						Highlight: false,
					}.Build(),
				}.Build(),
			}.Build(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: false,
		},
		{
			name: "negated prefix query does match a string with a different prefix",
			query: v1.Query_builder{
				BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{
						Field:     "Image",
						Value:     "!abcd",
						Highlight: false,
					}.Build(),
				}.Build(),
			}.Build(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: true,
		},
		{
			name: "negated regex query does not match a matching string",
			query: search.NewQueryBuilder().
				AddStrings(search.ImageName, search.NegateQueryString(search.RegexQueryString("av.*"))).
				ProtoQuery(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: false,
		},
		{
			name: "negated regex query matches a different string",
			query: search.NewQueryBuilder().
				AddStrings(search.ImageName, search.NegateQueryString(search.RegexQueryString("abcd.*"))).
				ProtoQuery(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: true,
		},
		{
			name: "negated map query returns true if all map entries match",
			query: search.NewQueryBuilder().
				AddMapQuery(search.ImageLabel, search.RegexQueryString("label.*"), search.NegateQueryString(search.RegexQueryString("zzz.*"))).
				ProtoQuery(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: true,
		},
		{
			name: "map query returns true if any map entry matches",
			query: search.NewQueryBuilder().
				AddMapQuery(search.ImageLabel, search.ExactMatchString("labelOne"), search.ExactMatchString("test.label.one")).
				ProtoQuery(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: true,
		},
		{
			name: "map query returns false if no map entries match",
			query: search.NewQueryBuilder().
				AddMapQuery(search.ImageLabel, search.ExactMatchString("zzz"), search.ExactMatchString("zzz")).
				ProtoQuery(),
			factory:     imageFactory,
			object:      passingImage,
			expectation: false,
		},
		{
			name: "map query returns false for an empty map",
			query: search.NewQueryBuilder().
				AddMapQuery(search.DeploymentLabel, search.ExactMatchString("key"), search.ExactMatchString("value")).
				ProtoQuery(),
			factory:     deploymentFactory,
			object:      deployment,
			expectation: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pred, err := c.factory.GeneratePredicate(c.query)
			require.NoError(t, err)
			require.NotNil(t, pred)

			assert.Equal(t, c.expectation, pred.Matches(c.object))
		})
	}
}

func TestSearchPredicateWithEnums(t *testing.T) {
	policyFactory := NewFactory("policy", &storage.Policy{})

	// Pass the predicate
	testPolicy := &storage.Policy{}
	testPolicy.SetId("p1")
	testPolicy.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_BUILD,
	})

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
			name:        "enums by name",
			query:       search.NewQueryBuilder().AddStrings(search.LifecycleStage, "BUILD").ProtoQuery(),
			expectation: true,
		},
		{
			name:        "enums by name fail",
			query:       search.NewQueryBuilder().AddStrings(search.LifecycleStage, "RUNTIME").ProtoQuery(),
			expectation: false,
		},
		{
			name:        "enums with comparator by name",
			query:       search.NewQueryBuilder().AddStrings(search.LifecycleStage, "<RUNTIME").ProtoQuery(),
			expectation: true,
		},
		{
			name:        "enums with comparator by name fail",
			query:       search.NewQueryBuilder().AddStrings(search.LifecycleStage, "<DEPLOY").ProtoQuery(),
			expectation: false,
		},
		{
			name: "handles any casing",
			query: v1.Query_builder{
				BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: "LifeCYCLE staGE", Value: "<RUNTIME"}.Build(),
				}.Build(),
			}.Build(),
			expectation: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pred, err := policyFactory.GeneratePredicate(c.query)
			assert.NotNil(t, pred)
			assert.NoError(t, err)
			assert.Equal(t, c.expectation, pred.Matches(testPolicy))
		})
	}
}

func TestSimplifications_AlwaysTrue(t *testing.T) {
	positive := []internalPredicate{
		andOf(),
		alwaysTrue,
		andOf(alwaysTrue, alwaysTrue),
		orOf(alwaysTrue, alwaysFalse),
		createLinkedStructPredicate(alwaysTrue, alwaysTrue),
	}

	for _, ip := range positive {
		assert.True(t, AlwaysTrue == wrapInternal(ip))
	}

	negative := []internalPredicate{
		orOf(),
		orOf(alwaysFalse, alwaysFalse),
		andOf(alwaysTrue, alwaysFalse),
		createMapLinkedPredicate(alwaysTrue),
		createSliceLinkedPredicate(),
	}

	for _, ip := range negative {
		assert.True(t, AlwaysTrue != wrapInternal(ip))
	}
}

func TestSimplifications_AlwaysFalse(t *testing.T) {
	positive := []internalPredicate{
		orOf(),
		alwaysFalse,
		andOf(alwaysTrue, alwaysFalse),
		orOf(alwaysFalse, alwaysFalse),
		createLinkedStructPredicate(alwaysTrue, alwaysFalse),
	}

	for _, ip := range positive {
		assert.True(t, AlwaysFalse == wrapInternal(ip))
	}

	negative := []internalPredicate{
		andOf(),
		andOf(alwaysTrue, alwaysTrue),
		orOf(alwaysTrue, alwaysFalse),
		createMapLinkedPredicate(alwaysFalse),
		createSliceLinkedPredicate(),
	}

	for _, ip := range negative {
		assert.True(t, AlwaysFalse != wrapInternal(ip))
	}
}
