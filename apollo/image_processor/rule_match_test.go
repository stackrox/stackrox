package imageprocessor

import (
	"regexp"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyRegexImageRule() *regexImageRule {
	return &regexImageRule{}
}

func getTestImage() *v1.Image {
	return &v1.Image{
		Scan: &v1.ImageScan{
			ScanTime: ptypes.TimestampNow(),
			Layers: []*v1.ScanLayer{
				{
					Components: []*v1.ImageScanComponents{
						{
							Name: "berkeleyDB",
							Vulns: []*v1.Vulnerability{
								{
									Cvss: 2,
									Cve:  "CVE-2016-1",
								},
								{
									Cvss: 4,
									Cve:  "CVE-2017-1",
								},
							},
						},
						{
							Name: "berkeleyCE",
							Vulns: []*v1.Vulnerability{
								{
									Cvss: 6,
								},
							},
						},
					},
				},
			},
		},
		Metadata: &v1.ImageMetadata{
			Created: ptypes.TimestampNow(),
			Layers: []*v1.ImageLayer{
				{
					Instruction: "CMD",
					Value:       "sudo bash",
				},
				{
					Instruction: "ENTRYPOINT",
					Value:       "executable",
				},
			},
		},
	}
}

func TestMatchComponent(t *testing.T) {
	violations, exists := emptyRegexImageRule().matchComponent(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	berkeleyRule := &regexImageRule{
		Component: regexp.MustCompile("^berkeley*"),
	}
	violations, exists = berkeleyRule.matchComponent(image)
	assert.True(t, exists)
	assert.Equal(t, 2, len(violations))
	berkeleyDBRule := &regexImageRule{
		Component: regexp.MustCompile("^berkeleyD.*"),
	}
	violations, exists = berkeleyDBRule.matchComponent(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))
}

func TestMatchLineRule(t *testing.T) {
	violations, exists := emptyRegexImageRule().matchLineRule(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	rule := &regexImageRule{
		LineRule: &lineRuleFieldRegex{
			Instruction: "CMD",
			Value:       regexp.MustCompile("^sudo.*"),
		},
	}
	violations, exists = rule.matchLineRule(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))
}

func TestMatchRuleToImageName(t *testing.T) {
	violations, exists := emptyRegexImageRule().matchRuleToImageName(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))
	rule := &regexImageRule{
		ImageNameRule: &imageNameRuleRegex{
			Registry:  regexp.MustCompile("^docker.io$"),
			Namespace: regexp.MustCompile("^library$"),
			Repo:      regexp.MustCompile("^nginx$"),
			Tag:       regexp.MustCompile("^latest$"),
		},
	}
	image := types.GenerateImageFromString("nginx")
	violations, exists = rule.matchRuleToImageName(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// If the image is totally different don't match
	image = types.GenerateImageFromString("summarizer")
	violations, exists = rule.matchRuleToImageName(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// If one of the values doesn't match then don't return a violation. Image parameters are AND'd together
	rule.ImageNameRule.Registry = regexp.MustCompile("^docker-registry$")
	image = types.GenerateImageFromString("nginx")
	violations, exists = rule.matchRuleToImageName(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))
}

func createTestRule(comparator v1.Comparator, op v1.MathOP, value float32) *regexImageRule {
	return &regexImageRule{
		CVSS: &v1.NumericalRule{
			Op:     comparator,
			MathOp: op,
			Value:  value,
		},
	}
}

func TestMatchCVSS(t *testing.T) {
	// CVSS is empty
	violations, exists := emptyRegexImageRule().matchCVSS(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	// AVG with <, =, >
	testRule := createTestRule(v1.Comparator_LESS_THAN, v1.MathOP_AVG, 5)
	violations, exists = testRule.matchCVSS(image)
	assert.Equal(t, 1, len(violations))

	testRule = createTestRule(v1.Comparator_EQUALS, v1.MathOP_AVG, 4)
	violations, exists = testRule.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	testRule = createTestRule(v1.Comparator_GREATER_THAN, v1.MathOP_AVG, 3)
	violations, exists = testRule.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// Don't fire if not equal
	testRule = createTestRule(v1.Comparator_EQUALS, v1.MathOP_AVG, 3)
	violations, exists = testRule.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// MIN with =
	testRule = createTestRule(v1.Comparator_EQUALS, v1.MathOP_MIN, 2)
	violations, exists = testRule.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// MAX with =
	testRule = createTestRule(v1.Comparator_EQUALS, v1.MathOP_MAX, 6)
	violations, exists = testRule.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))
}

func TestMatchCVE(t *testing.T) {
	// CVE is empty
	violations, exists := emptyRegexImageRule().matchCVE(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	rule := &regexImageRule{
		CVE: regexp.MustCompile("^CVE-2016.*"),
	}
	violations, exists = rule.matchCVE(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	rule = &regexImageRule{
		CVE: regexp.MustCompile("^CVE-2018.*"),
	}
	violations, exists = rule.matchCVE(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))
}

func TestMatchImageAge(t *testing.T) {
	// Image Age is empty
	violations, exists := emptyRegexImageRule().matchImageAge(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	now := ptypes.TimestampNow()

	// Does not violate
	rule := &regexImageRule{
		ImageAgeDays: 1,
	}
	image.Metadata.Created = now
	violations, exists = rule.matchImageAge(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// Violates
	protoTS, _ := ptypes.TimestampProto(time.Now().AddDate(0, 0, -2))
	image.Metadata.Created = protoTS
	violations, exists = rule.matchImageAge(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))
}

func TestMatchScanAge(t *testing.T) {
	// Scan Age is empty
	violations, exists := emptyRegexImageRule().matchScanAge(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()
	now := ptypes.TimestampNow()

	// Does not violate
	rule := &regexImageRule{
		ScanAgeDays: 1,
	}
	image.Scan.ScanTime = now
	violations, exists = rule.matchScanAge(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// Violates
	protoTS, err := ptypes.TimestampProto(time.Now().AddDate(0, 0, -2))
	require.Nil(t, err)
	image.Scan.ScanTime = protoTS
	violations, exists = rule.matchScanAge(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))
}

func TestMatchRuleToImage(t *testing.T) {
	// If empty then no violations
	alert := emptyRegexImageRule().matchRuleToImage(nil)
	assert.Nil(t, alert)

	image := getTestImage()

	rule := &regexImageRule{
		LineRule: &lineRuleFieldRegex{
			Instruction: "CMD",
			Value:       regexp.MustCompile("^sudo.*"), // generates 1 violation
		},
		Component: regexp.MustCompile("^berkeley*"), // generates 2 violations
	}

	// Make sure if two are specified and both have violations that we receive the violations
	alert = rule.matchRuleToImage(image)
	assert.NotNil(t, alert)
	assert.Equal(t, 3, len(alert.Violations))

	// Make sure if two are specified, but one does not have a violation that we receive no violations
	rule.Component = regexp.MustCompile("^blah*") // should make ComponentMatch generate no violations so overall alert fails
	alert = rule.matchRuleToImage(image)
	assert.Nil(t, alert)
}
