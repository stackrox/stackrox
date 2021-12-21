package component

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/central/risk/scorer/vulns"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scancomponent"
)

const (
	vulnSaturation = 100
	vulnMaxScore   = 4
)

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in a component
type vulnerabilitiesMultiplier struct {
	typ     string
	heading string
}

// NewVulnerabilities provides a multiplier that scores the data based on the CVSS scores and number of CVEs
func NewVulnerabilities(typ, heading string) Multiplier {
	return &vulnerabilitiesMultiplier{
		typ:     typ,
		heading: heading,
	}
}

// Score takes a component and evaluates its risk based on vulnerabilities
func (c *vulnerabilitiesMultiplier) Score(_ context.Context, component scancomponent.ScanComponent) *storage.Risk_Result {
	log := logging.LoggerForModule()
	if strings.Contains(component.GetName(), "log4j") {
		log.Errorf("[Score - Component] STARTING scoring for component %s", component.GetName())
	}
	min, max, sum, numCVEs := vulns.ProcessComponent(component)
	if numCVEs == 0 {
		return nil
	}
	if strings.Contains(component.GetName(), "log4j") {
		log.Errorf("[Score - Component] ND scoring for component %s and got %v vulns", component.GetName(), numCVEs)
	}

	return &storage.Risk_Result{
		Name: c.heading,
		Factors: []*storage.Risk_Result_Factor{
			{
				Message: fmt.Sprintf("%s Component %s version %s contains %d CVEs with severities ranging between %s and %s",
					c.typ, component.GetName(), component.GetVersion(), numCVEs, min.Severity, max.Severity),
			},
		},
		Score: multipliers.NormalizeScore(sum, vulnSaturation, vulnMaxScore),
	}
}
