package main

import "github.com/quay/claircore"

// Fixtures are explicit native records. Expectations live separately in the
// Scanner and backend tests; this command never reads tests or a source bundle.
type operation struct {
	Member, Updater string
	Vulnerabilities []*claircore.Vulnerability
	Enrichments     []enrichmentFixture
}

type enrichmentFixture struct {
	Tags    []string
	Payload any
}

func fixtures() []operation {
	var out []operation
	out = append(out, alpineFixtures...)
	out = append(out, awsFixtures...)
	out = append(out, debianFixtures...)
	out = append(out, manualFixtures...)
	out = append(out, nvdFixtures...)
	out = append(out, oracleFixtures...)
	out = append(out, osvFixtures...)
	out = append(out, photonFixtures...)
	out = append(out, rhel_vexFixtures...)
	out = append(out, rhelUnaffectedFixtures...)
	out = append(out, stackrox_rhel_csafFixtures...)
	out = append(out, ubuntuFixtures...)
	return out
}
