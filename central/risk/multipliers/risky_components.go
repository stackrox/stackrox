package multipliers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// RiskyComponentCountHeading is the risk result name for scores calculated by this multiplier.
	RiskyComponentCountHeading = "Components Useful for Attackers"

	riskyComponentCountFloor = 0
	riskyComponentCountCeil  = 10
	maxRiskyScore            = 1.5
	maxComponentsInMessage   = 5
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
func (c *riskyComponentCountMultiplier) Score(deployment *storage.Deployment) *storage.Risk_Result {
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
	if largestRiskySet == nil || largestRiskySet.Cardinality() == 0 {
		return nil
	}

	// Linear increase between 10 components and 20 components from weight of 1 to 1.5.
	score := float32(1.0) + float32(largestRiskySet.Cardinality()-riskyComponentCountFloor)/float32(riskyComponentCountCeil-riskyComponentCountFloor)/float32(2)
	if score > maxRiskyScore {
		score = maxRiskyScore
	}

	return &storage.Risk_Result{
		Name: RiskyComponentCountHeading,
		Factors: []*storage.Risk_Result_Factor{
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
	if len(largestRiskySet) > maxComponentsInMessage {
		componentsInMessage := largestRiskySet[:5]
		componentsString := strings.Join(componentsInMessage, ", ")
		diff := len(largestRiskySet) - maxComponentsInMessage
		return fmt.Sprintf("an image contains components: %s and %d other(s) that are useful for attackers", componentsString, diff)
	}

	// Otherwise use all of the components in the message.
	componentsString := strings.Join(largestRiskySet, ", ")
	return fmt.Sprintf("an image contains component(s) useful for attackers: %s", componentsString)
}
