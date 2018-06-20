package imageprocessor

import (
	"regexp"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyRegexImagePolicy() *compiledImagePolicy {
	return &compiledImagePolicy{}
}

func getTestImage() *v1.Image {
	return &v1.Image{
		Scan: &v1.ImageScan{
			ScanTime: ptypes.TimestampNow(),
			Components: []*v1.ImageScanComponent{
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
					Version: "v1.5.0",
				},
				{
					Name: "berkeleyCE",
					Vulns: []*v1.Vulnerability{
						{
							Cvss: 6,
						},
					},
					Version: "v1.2.0",
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

func clearImagePluginData(i *v1.Image) {
	i.Metadata = nil
	i.Scan = nil
}

func TestMatchComponent(t *testing.T) {
	violations, exists := emptyRegexImagePolicy().matchComponent(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	componentRegex, err := compileComponent(&v1.Component{
		Name:    "^berkeley*",
		Version: ".*",
	})
	assert.NoError(t, err)

	berkeleyPolicy := &compiledImagePolicy{
		Component: componentRegex,
	}
	violations, exists = berkeleyPolicy.matchComponent(image)
	assert.True(t, exists)
	assert.Equal(t, 2, len(violations))

	componentRegex, err = compileComponent(&v1.Component{
		Name:    "^berkeleyD.*",
		Version: "^v1.5.0$",
	})
	assert.NoError(t, err)

	berkeleyDBPolicy := &compiledImagePolicy{
		Component: componentRegex,
	}
	violations, exists = berkeleyDBPolicy.matchComponent(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// Check if there is no metadata
	clearImagePluginData(image)
	violations, exists = berkeleyDBPolicy.matchComponent(image)
	assert.Nil(t, violations)
	assert.True(t, exists)
}

func TestMatchLineRule(t *testing.T) {
	violations, exists := emptyRegexImagePolicy().matchLineRule(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	policy := &compiledImagePolicy{
		LineRule: &lineRuleFieldRegex{
			Instruction: "CMD",
			Value:       regexp.MustCompile("^sudo.*"),
		},
	}
	violations, exists = policy.matchLineRule(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// Check if there is no scan
	clearImagePluginData(image)
	violations, exists = policy.matchLineRule(image)
	assert.Nil(t, violations)
	assert.True(t, exists)
}

func TestMatchImageName(t *testing.T) {
	violations, exists := emptyRegexImagePolicy().matchImageName(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))
	policy := &compiledImagePolicy{
		ImageNamePolicy: &imageNamePolicyRegex{
			Registry:  regexp.MustCompile("^docker.io$"),
			Namespace: regexp.MustCompile("^library$"),
			Repo:      regexp.MustCompile("^nginx$"),
			Tag:       regexp.MustCompile("^latest$"),
		},
	}
	image := images.GenerateImageFromString("nginx")
	violations, exists = policy.matchImageName(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// If the image is totally different don't match
	image = images.GenerateImageFromString("summarizer")
	violations, exists = policy.matchImageName(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// If one of the values doesn't match then don't return a violation. Image parameters are AND'd together
	policy.ImageNamePolicy.Registry = regexp.MustCompile("^docker-registry$")
	image = images.GenerateImageFromString("nginx")
	violations, exists = policy.matchImageName(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	policy.ImageNamePolicy.Registry = regexp.MustCompile("^stackrox.io$")
	policy.ImageNamePolicy.Namespace = regexp.MustCompile("^prevent")
	image = images.GenerateImageFromString("stackrox.io/prevent:latest")
	violations, exists = policy.matchImageName(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))
}

func createTestPolicy(comparator v1.Comparator, op v1.MathOP, value float32) *compiledImagePolicy {
	return &compiledImagePolicy{
		CVSS: &v1.NumericalPolicy{
			Op:     comparator,
			MathOp: op,
			Value:  value,
		},
	}
}

func TestMatchCVSS(t *testing.T) {
	// CVSS is empty
	violations, exists := emptyRegexImagePolicy().matchCVSS(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	// AVG with <, =, >
	testPolicy := createTestPolicy(v1.Comparator_LESS_THAN, v1.MathOP_AVG, 5)
	violations, exists = testPolicy.matchCVSS(image)
	assert.Equal(t, 1, len(violations))

	testPolicy = createTestPolicy(v1.Comparator_EQUALS, v1.MathOP_AVG, 4)
	violations, exists = testPolicy.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	testPolicy = createTestPolicy(v1.Comparator_GREATER_THAN, v1.MathOP_AVG, 3)
	violations, exists = testPolicy.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// Don't fire if not equal
	testPolicy = createTestPolicy(v1.Comparator_EQUALS, v1.MathOP_AVG, 3)
	violations, exists = testPolicy.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// MIN with =
	testPolicy = createTestPolicy(v1.Comparator_EQUALS, v1.MathOP_MIN, 2)
	violations, exists = testPolicy.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// MAX with =
	testPolicy = createTestPolicy(v1.Comparator_EQUALS, v1.MathOP_MAX, 6)
	violations, exists = testPolicy.matchCVSS(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// Check if there is no scan
	clearImagePluginData(image)
	violations, exists = testPolicy.matchCVSS(image)
	assert.Nil(t, violations)
	assert.True(t, exists)
}

func TestMatchCVE(t *testing.T) {
	// CVE is empty
	violations, exists := emptyRegexImagePolicy().matchCVE(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()

	policy := &compiledImagePolicy{
		CVE: regexp.MustCompile("^CVE-2016.*"),
	}
	violations, exists = policy.matchCVE(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	policy = &compiledImagePolicy{
		CVE: regexp.MustCompile("^CVE-2018.*"),
	}
	violations, exists = policy.matchCVE(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// Check if there is no scan
	clearImagePluginData(image)
	violations, exists = policy.matchCVE(image)
	assert.Nil(t, violations)
	assert.True(t, exists)
}

func TestMatchImageAge(t *testing.T) {
	// Image Age is empty
	violations, exists := emptyRegexImagePolicy().matchImageAge(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()
	now := ptypes.TimestampNow()

	// Does not violate and does not exist
	policy := &compiledImagePolicy{
		ImageAgeDays: nil,
	}
	violations, exists = policy.matchImageAge(image)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	// Does not violate and exists
	var days int64 = 1
	policy = &compiledImagePolicy{
		ImageAgeDays: &days,
	}
	image.Metadata.Created = now
	violations, exists = policy.matchImageAge(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// Violates
	protoTS, _ := ptypes.TimestampProto(time.Now().AddDate(0, 0, -2))
	image.Metadata.Created = protoTS
	violations, exists = policy.matchImageAge(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// Check if there is no metadata
	clearImagePluginData(image)
	violations, exists = policy.matchImageAge(image)
	assert.Nil(t, violations)
	assert.True(t, exists)
}

func TestMatchScanAge(t *testing.T) {
	// Scan Age is empty
	violations, exists := emptyRegexImagePolicy().matchScanAge(nil)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	image := getTestImage()
	now := ptypes.TimestampNow()

	// Does not violate and does not exist
	policy := &compiledImagePolicy{
		ScanAgeDays: nil,
	}
	violations, exists = policy.matchScanAge(image)
	assert.False(t, exists)
	assert.Equal(t, 0, len(violations))

	// Does not violate and exists
	var days int64 = 1
	policy = &compiledImagePolicy{
		ScanAgeDays: &days,
	}
	image.Scan.ScanTime = now
	violations, exists = policy.matchScanAge(image)
	assert.True(t, exists)
	assert.Equal(t, 0, len(violations))

	// Violates
	protoTS, err := ptypes.TimestampProto(time.Now().AddDate(0, 0, -2))
	require.Nil(t, err)
	image.Scan.ScanTime = protoTS
	violations, exists = policy.matchScanAge(image)
	assert.True(t, exists)
	assert.Equal(t, 1, len(violations))

	// Check if there is no scan
	clearImagePluginData(image)
	violations, exists = policy.matchScanAge(image)
	assert.Nil(t, violations)
	assert.True(t, exists)
}

func TestMatchPolicyToImage(t *testing.T) {
	// If empty then no violations
	violations, valid := emptyRegexImagePolicy().Match(nil, &v1.Container{})
	assert.False(t, valid)
	assert.Nil(t, violations)

	image := getTestImage()

	componentRegex, err := compileComponent(&v1.Component{
		Name:    "^berkeley*",
		Version: ".*",
	})
	assert.NoError(t, err)

	policy := &compiledImagePolicy{
		Original: &v1.Policy{
			Name:     "policy1",
			Severity: v1.Severity_CRITICAL_SEVERITY,
		},
		LineRule: &lineRuleFieldRegex{
			Instruction: "CMD",
			Value:       regexp.MustCompile("^sudo.*"), // generates 1 violation
		},
		Component: componentRegex, // generates 2 violations
	}

	// Make sure if two are specified and both have violations that we receive the violations
	violations, valid = policy.Match(nil, &v1.Container{Image: image})
	assert.True(t, valid)
	assert.NotNil(t, violations)
	assert.Equal(t, 3, len(violations))

	componentRegex, err = compileComponent(&v1.Component{
		Name:    "^blah*",
		Version: ".*",
	})
	assert.NoError(t, err)

	// Make sure if two are specified, but one does not have a violation that we receive no violations
	policy.Component = componentRegex // should make ComponentMatch generate no violations so overall alert fails
	violations, _ = policy.Match(nil, &v1.Container{Image: image})
	assert.Nil(t, violations)
}
