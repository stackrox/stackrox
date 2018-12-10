package multipliers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// RiskyComponentCountHeading is the risk result name for scores calculated by this multiplier.
	RiskyComponentCountHeading = "Components Useful for Attackers"

	componentCountFloor     = 0
	componentCountCeil      = 10
	maxScore                = 1.5
	maxComnponentsInMessage = 5
)

var riskyComponents = set.NewStringSet(
	"apk",
	"apt",
	"bash",
	"curl",
	"dnf",
	"netcat",
	"nmap",
	"rpm",
	"sh",
	"tcsh",
	"telnet",
	"wget",
	"yum",
)

// riskyComponentCountMultiplier is a scorer for the components in an image that can be used by attackers.
type riskyComponentCountMultiplier struct{}

// NewRiskyComponents provides a multiplier that scores the data based on the the number of risky components in images.
func NewRiskyComponents() Multiplier {
	return &riskyComponentCountMultiplier{}
}

// Score takes a deployment and evaluates its risk based on image component counts.
func (c *riskyComponentCountMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	// Get the largest number of risky components in an image
	var largestRiskySet *set.StringSet
	for _, container := range deployment.GetContainers() {
		// Create a name to version map of all the image components.
		presentComponents := set.NewStringSet()
		for _, component := range container.GetImage().GetScan().GetComponents() {
			presentComponents.Add(component.GetName())
		}

		// Count how many known risky components match a labeled component.
		riskySet := riskyComponents.Intersect(presentComponents)

		// Keep track of the image with the largest number of risky components.
		if largestRiskySet == nil || riskySet.Cardinality() > largestRiskySet.Cardinality() {
			largestRiskySet = &riskySet
		}
	}
	if largestRiskySet == nil {
		return nil
	}

	// Linear increase between 10 components and 20 components from weight of 1 to 1.5.
	score := float32(1.0) + float32(largestRiskySet.Cardinality()-componentCountFloor)/float32(componentCountCeil-componentCountFloor)/float32(2)
	if score > maxScore {
		score = maxScore
	}

	return &v1.Risk_Result{
		Name: RiskyComponentCountHeading,
		Factors: []*v1.Risk_Result_Factor{
			{Message: generateMessage(largestRiskySet.AsSlice())},
		},
		Score: score,
	}
}

func generateMessage(largestRiskySet []string) string {
	// Sort for message stability.
	sort.SliceStable(largestRiskySet, func(i, j int) bool {
		return largestRiskySet[i] < largestRiskySet[j]
	})

	// If we have more than 5 risky components, prune the message.
	if len(largestRiskySet) > 5 {
		componentsInMessage := largestRiskySet[:5]
		componentsString := strings.Join(componentsInMessage, ", ")
		diff := len(largestRiskySet) - 5
		return fmt.Sprintf("an image contains components: %s and %d other(s) that are useful for attackers", componentsString, diff)
	}

	// Otherwise use all of the components in the message.
	componentsString := strings.Join(largestRiskySet, ", ")
	return fmt.Sprintf("an image contains component(s) useful for attackers: %s", componentsString)
}
