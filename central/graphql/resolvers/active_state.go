package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
)

// ActiveStateEnum represents the active state of a vuln or component in a deployment.
//
//go:generate stringer -type=ActiveStateEnum
type ActiveStateEnum int32

const (
	// Undetermined means activeness cannot be determined.
	Undetermined ActiveStateEnum = iota
	// Inactive means the vulnerability or component is not active.
	Inactive
	// Active means the vulnerability or component is active.
	Active
	// FeatureDisabled means the feature is disabled
	FeatureDisabled
)

func init() {
	schema := getBuilder()

	utils.Must(schema.AddType("ActiveState", []string{
		"state: String!",
		"activeContexts: [ActiveComponent_ActiveContext!]!",
	}))
}

type activeStateResolver struct {
	root               *Resolver
	state              ActiveStateEnum
	activeComponentIDs []string
	imageScope         string
}

// State is the activeness state
func (asr *activeStateResolver) State(_ context.Context) string {
	if !features.ActiveVulnMgmt.Enabled() {
		return FeatureDisabled.String()
	}
	return asr.state.String()
}

// ActiveContexts is the slice of active contexts in deployment
func (asr *activeStateResolver) ActiveContexts(ctx context.Context) ([]*activeComponent_ActiveContextResolver, error) {
	var acs []*activeComponent_ActiveContextResolver
	if !features.ActiveVulnMgmt.Enabled() {
		return acs, nil
	}
	if len(asr.activeComponentIDs) == 0 {
		return acs, nil
	}
	activeComponents, err := asr.root.ActiveComponent.GetBatch(ctx, asr.activeComponentIDs)
	if err != nil {
		return nil, err
	}
	for _, activeComponent := range activeComponents {
		for _, ac := range activeComponent.GetActiveContextsSlice() {
			if asr.imageScope == "" || ac.ImageId == asr.imageScope {
				acs = append(acs, &activeComponent_ActiveContextResolver{ctx: ctx, data: ac})
			}
		}
	}
	return acs, nil
}
