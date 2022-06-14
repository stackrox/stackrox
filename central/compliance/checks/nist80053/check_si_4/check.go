package checksi4

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const (
	controlID = `NIST_SP_800_53_Rev_4:SI_4`

	interpretationText = `This control requires the deployment of security monitoring tools in information systems to detect potential attacks or unauthorized connections.

For this control, StackRox verifies that StackRox monitoring components are deployed into, and successfully operating in, each cluster.`

	neverCheckedInEvidence = `The StackRox Sensor for this cluster has never checked in (as of the time of this compliance assessment).`
)

func checkClusterCheckedInInThePastHour(ctx framework.ComplianceContext) {
	lastContact := ctx.Data().Cluster().GetHealthStatus().GetLastContact()
	if lastContact == nil {
		framework.Fail(ctx, neverCheckedInEvidence)
		return
	}
	lastContactGoTime, err := types.TimestampFromProto(lastContact)
	// This should basically never happen, but the best we can do here is to treat it as a non-existent
	// timestamp.
	if err != nil {
		framework.Fail(ctx, neverCheckedInEvidence)
		return
	}
	if lastContactGoTime.Before(time.Now().Add(-1 * time.Hour)) {
		framework.Failf(ctx, "The StackRox Sensor for this cluster has not checked in for the past hour (last check-in as of the time of this compliance assessment: %s).", lastContactGoTime)
	} else {
		framework.Passf(ctx, "The StackRox Sensor for this cluster has checked in during the past hour (last check-in as of the time of this compliance assessment: %s).", lastContactGoTime)
	}
}

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Cluster"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			framework.Pass(ctx, "The StackRox Kubernetes Security Platform is installed, and provides information system monitoring.")
			checkClusterCheckedInInThePastHour(ctx)
		})
}
