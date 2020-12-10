package resolvers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlottedVulnerabilities", []string{
			"basicVulnCounter: VulnerabilityCounter!",
			"vulns(pagination: Pagination): [EmbeddedVulnerability]!",
		}),
	)
}

// PlottedVulnerabilitiesResolver returns the data required by top risky entity scatter-plot on vuln mgmt dashboard
type PlottedVulnerabilitiesResolver struct {
	root    *Resolver
	all     []string
	fixable []string
	// TODO: Delete once node mock API is deleted.
	mock bool
}

func newPlottedVulnerabilitiesResolver(ctx context.Context, root *Resolver, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	q = tryUnsuppressedQuery(q)
	all, err := root.CVEDataStore.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	fixable, err := root.CVEDataStore.Search(ctx,
		search.NewConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	return &PlottedVulnerabilitiesResolver{
		root:    root,
		all:     search.ResultsToIDs(all),
		fixable: search.ResultsToIDs(fixable),
	}, nil
}

// BasicVulnCounter returns the vulnCounter for scatter-plot with only total and fixable
func (pvr *PlottedVulnerabilitiesResolver) BasicVulnCounter(ctx context.Context) (*VulnerabilityCounterResolver, error) {
	return &VulnerabilityCounterResolver{
		all: &VulnerabilityFixableCounterResolver{
			total:   int32(len(pvr.all)),
			fixable: int32(len(pvr.fixable)),
		},
	}, nil
}

// Vulns returns the vulns for scatter-plot
func (pvr *PlottedVulnerabilitiesResolver) Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	if pvr.mock {
		return pvr.mockVulns()
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	if len(pvr.all) == 0 {
		return nil, nil
	}

	pagination := q.GetPagination()
	q = search.NewQueryBuilder().AddDocIDs(pvr.all...).ProtoQuery()
	q.Pagination = pagination

	paginatedVulns, err := pvr.root.CVEDataStore.SearchRawCVEs(ctx, q)
	if err != nil {
		return nil, err
	}

	vulns := make([]VulnerabilityResolver, 0, len(paginatedVulns))
	for _, vuln := range paginatedVulns {
		vulns = append(vulns, &cVEResolver{root: pvr.root, data: vuln})
	}
	return vulns, nil
}

func (pvr *PlottedVulnerabilitiesResolver) mockVulns() ([]VulnerabilityResolver, error) {
	return []VulnerabilityResolver{
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-0",
				Cvss:         9.9,
				ImpactScore:  6.0,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "The Kubelet and kube-proxy components in versions 1.1.0-1.16.10, 1.17.0-1.17.6, and 1.18.0-1.18.3 were found to contain a security issue which allows adjacent hosts to reach TCP and UDP services bound to 127.0.0.1 running on the node or in the node's network namespace. Such a service is generally thought to be reachable only by other processes on the same host, but due to this defeect, could be reachable by other hosts on the same LAN as the node, or by containers running on the same node as the service.",
				Link:         "https://github.com/kubernetes/kubernetes/issues/92315",
				ScoreVersion: storage.CVE_V3,
				CvssV2: &storage.CVSSV2{
					Vector:              "AV:A/AC:L/Au:N/C:P/I:P/A:P",
					AttackVector:        storage.CVSSV2_ATTACK_ADJACENT,
					AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
					Authentication:      storage.CVSSV2_AUTH_NONE,
					Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
					Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
					Availability:        storage.CVSSV2_IMPACT_PARTIAL,
					ExploitabilityScore: 6.5,
					ImpactScore:         6.4,
					Score:               5.8,
					Severity:            storage.CVSSV2_MEDIUM,
				},
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:C/C:H/I:H/A:H",
					ExploitabilityScore: 3.1,
					ImpactScore:         6.0,
					AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
					UserInteraction:     storage.CVSSV3_UI_NONE,
					Scope:               storage.CVSSV3_CHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
					Integrity:           storage.CVSSV3_IMPACT_HIGH,
					Availability:        storage.CVSSV3_IMPACT_HIGH,
					Score:               9.9,
					Severity:            storage.CVSSV3_CRITICAL,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-1",
				Cvss:         8.0,
				ImpactScore:  6.0,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "one",
				Link:         "https://github.com/kubernetes/kubernetes/issues/1",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:A/AC:H/PR:L/UI:N/S:C/C:H/I:H/A:H",
					ExploitabilityScore: 1.3,
					ImpactScore:         6.0,
					AttackVector:        storage.CVSSV3_ATTACK_ADJACENT,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
					UserInteraction:     storage.CVSSV3_UI_NONE,
					Scope:               storage.CVSSV3_CHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
					Integrity:           storage.CVSSV3_IMPACT_HIGH,
					Availability:        storage.CVSSV3_IMPACT_HIGH,
					Score:               8.0,
					Severity:            storage.CVSSV3_HIGH,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-2",
				Cvss:         9.0,
				ImpactScore:  6.0,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "two",
				Link:         "https://github.com/kubernetes/kubernetes/issues/2",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:A/AC:L/PR:L/UI:N/S:C/C:H/I:H/A:H",
					ExploitabilityScore: 2.3,
					ImpactScore:         6.0,
					AttackVector:        storage.CVSSV3_ATTACK_ADJACENT,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
					UserInteraction:     storage.CVSSV3_UI_NONE,
					Scope:               storage.CVSSV3_CHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
					Integrity:           storage.CVSSV3_IMPACT_HIGH,
					Availability:        storage.CVSSV3_IMPACT_HIGH,
					Score:               9.0,
					Severity:            storage.CVSSV3_CRITICAL,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-3",
				Cvss:         9.0,
				ImpactScore:  6.0,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "three",
				Link:         "https://github.com/kubernetes/kubernetes/issues/3",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:A/AC:L/PR:L/UI:N/S:C/C:H/I:H/A:H",
					ExploitabilityScore: 2.3,
					ImpactScore:         6.0,
					AttackVector:        storage.CVSSV3_ATTACK_ADJACENT,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
					UserInteraction:     storage.CVSSV3_UI_NONE,
					Scope:               storage.CVSSV3_CHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
					Integrity:           storage.CVSSV3_IMPACT_HIGH,
					Availability:        storage.CVSSV3_IMPACT_HIGH,
					Score:               9.0,
					Severity:            storage.CVSSV3_CRITICAL,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-4",
				Cvss:         6.5,
				ImpactScore:  5.9,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "four",
				Link:         "https://github.com/kubernetes/kubernetes/issues/4",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:L/AC:L/PR:H/UI:R/S:U/C:H/I:H/A:H",
					ExploitabilityScore: 0.6,
					ImpactScore:         5.9,
					AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_HIGH,
					UserInteraction:     storage.CVSSV3_UI_REQUIRED,
					Scope:               storage.CVSSV3_UNCHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
					Integrity:           storage.CVSSV3_IMPACT_HIGH,
					Availability:        storage.CVSSV3_IMPACT_HIGH,
					Score:               6.5,
					Severity:            storage.CVSSV3_MEDIUM,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-5",
				Cvss:         5.8,
				ImpactScore:  5.2,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "five",
				Link:         "https://github.com/kubernetes/kubernetes/issues/5",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:L/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:H",
					ExploitabilityScore: 0.6,
					ImpactScore:         5.2,
					AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_HIGH,
					UserInteraction:     storage.CVSSV3_UI_REQUIRED,
					Scope:               storage.CVSSV3_UNCHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_NONE,
					Integrity:           storage.CVSSV3_IMPACT_HIGH,
					Availability:        storage.CVSSV3_IMPACT_HIGH,
					Score:               5.8,
					Severity:            storage.CVSSV3_MEDIUM,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-6",
				Cvss:         2.0,
				ImpactScore:  1.4,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "six",
				Link:         "https://github.com/kubernetes/kubernetes/issues/6",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:L/AC:L/PR:H/UI:R/S:U/C:N/I:L/A:N",
					ExploitabilityScore: 0.6,
					ImpactScore:         1.4,
					AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_HIGH,
					UserInteraction:     storage.CVSSV3_UI_REQUIRED,
					Scope:               storage.CVSSV3_UNCHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_NONE,
					Integrity:           storage.CVSSV3_IMPACT_LOW,
					Availability:        storage.CVSSV3_IMPACT_NONE,
					Score:               2.0,
					Severity:            storage.CVSSV3_LOW,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-7",
				Cvss:         2.8,
				ImpactScore:  1.4,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "seven",
				Link:         "https://github.com/kubernetes/kubernetes/issues/7",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:L/AC:L/PR:H/UI:R/S:C/C:N/I:L/A:N",
					ExploitabilityScore: 1.1,
					ImpactScore:         1.4,
					AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_HIGH,
					UserInteraction:     storage.CVSSV3_UI_REQUIRED,
					Scope:               storage.CVSSV3_CHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_NONE,
					Integrity:           storage.CVSSV3_IMPACT_LOW,
					Availability:        storage.CVSSV3_IMPACT_NONE,
					Score:               2.0,
					Severity:            storage.CVSSV3_LOW,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-8",
				Cvss:         3.2,
				ImpactScore:  1.4,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "eight",
				Link:         "https://github.com/kubernetes/kubernetes/issues/8",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:L/AC:L/PR:L/UI:R/S:C/C:N/I:L/A:N",
					ExploitabilityScore: 1.5,
					ImpactScore:         1.4,
					AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
					UserInteraction:     storage.CVSSV3_UI_REQUIRED,
					Scope:               storage.CVSSV3_CHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_NONE,
					Integrity:           storage.CVSSV3_IMPACT_LOW,
					Availability:        storage.CVSSV3_IMPACT_NONE,
					Score:               3.2,
					Severity:            storage.CVSSV3_LOW,
				},
			},
		},
		&cVEResolver{
			root: pvr.root,
			data: &storage.CVE{
				Id:           "CVE-2020-9",
				Cvss:         2.8,
				ImpactScore:  1.4,
				Type:         storage.CVE_NODE_CVE,
				Summary:      "nine",
				Link:         "https://github.com/kubernetes/kubernetes/issues/9",
				ScoreVersion: storage.CVE_V3,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:L/AC:L/PR:L/UI:R/S:U/C:N/I:L/A:N",
					ExploitabilityScore: 1.3,
					ImpactScore:         1.4,
					AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
					UserInteraction:     storage.CVSSV3_UI_REQUIRED,
					Scope:               storage.CVSSV3_UNCHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_NONE,
					Integrity:           storage.CVSSV3_IMPACT_LOW,
					Availability:        storage.CVSSV3_IMPACT_NONE,
					Score:               2.8,
					Severity:            storage.CVSSV3_LOW,
				},
			},
		},
	}, nil
}
