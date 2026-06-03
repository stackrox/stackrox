package csaf

import (
	"regexp"
	"sync"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
)

// Advisory represents a CSAF security advisory.
type Advisory struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ReleaseDate time.Time `json:"release_date"`
	Severity    string    `json:"severity"`
	CVSSv3      CVSSScore `json:"cvssv3"`
	CVSSv2      CVSSScore `json:"cvssv2"`
}

// CVSSScore represents a CVSS score with its vector string.
type CVSSScore struct {
	BaseScore float64 `json:"base_score"`
	Vector    string  `json:"vector"`
}

// Enricher enriches vulnerability reports with CSAF advisory data.
type Enricher struct {
	mu         sync.RWMutex
	advisories map[string]*Advisory
}

// Option configures an Enricher.
type Option func(*Enricher)

// NewEnricher creates a new CSAF enricher.
func NewEnricher(opts ...Option) *Enricher {
	e := &Enricher{
		advisories: make(map[string]*Advisory),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// WithStaticAdvisories configures the enricher with a static set of advisories.
func WithStaticAdvisories(advisories map[string]*Advisory) Option {
	return func(e *Enricher) {
		e.advisories = advisories
	}
}

// SetAdvisories updates the advisories map used for enrichment.
// Thread-safe for concurrent use.
func (e *Enricher) SetAdvisories(advisories map[string]*Advisory) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.advisories = advisories
}

// rhsaPattern matches RHSA, RHBA, and RHEA advisory identifiers.
var rhsaPattern = regexp.MustCompile(`(RH[SBE]A-\d{4}:\d+)`)

// Enrich extracts RHSA/RHBA/RHEA names from vulnerability names and looks them up
// in the advisories map. Returns map[vulnID]*Advisory.
func (e *Enricher) Enrich(vr *clairclient.VulnerabilityReport) (map[string]*Advisory, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]*Advisory)

	for vulnID, vuln := range vr.Vulnerabilities {
		// Extract advisory name from vulnerability name
		matches := rhsaPattern.FindStringSubmatch(vuln.Name)
		if len(matches) < 2 {
			continue
		}

		advisoryName := matches[1]

		// Look up advisory
		if advisory, exists := e.advisories[advisoryName]; exists {
			result[vulnID] = advisory
		}
	}

	return result, nil
}
