package nvd

import "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"

var (
	// cvesWithoutFixedBy represents NVD CVEs that may not have correct fixed by version
	cvesWithoutFixedBy = map[string][]*schema.NVDCVEFeedJSON10DefNode{
		// https://github.com/kubernetes/kubernetes/issues/19479
		"CVE-2016-1905": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionEndExcluding: "1.2.0",
						Vulnerable:          true,
					},
				},
			},
		},
		// https://access.redhat.com/errata/RHSA-2016:0351
		// https://access.redhat.com/errata/RHSA-2016:0070
		"CVE-2016-1906": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:   "cpe:2.3:a:redhat:openshift_container_platform:3.0.0:*:*:*:*:*:*:*",
						Vulnerable: true,
					},
					{
						Cpe23Uri:   "cpe:2.3:a:redhat:openshift_container_platform:3.1.0:*:*:*:*:*:*:*",
						Vulnerable: true,
					},
				},
			},
		},
		// https://nvd.nist.gov/vuln/detail/CVE-2020-10712
		"CVE-2020-10712": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:              "cpe:2.3:a:redhat:openshift_container_platform:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "4.1",
						Vulnerable:            true,
					},
				},
			},
		},
		// https://github.com/kubernetes/kubernetes/issues/34517
		"CVE-2016-7075": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionEndExcluding: "1.2.7",
						Vulnerable:          true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.3.0",
						VersionEndExcluding:   "1.3.9",
						Vulnerable:            true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.4.0",
						VersionEndExcluding:   "1.4.3",
						Vulnerable:            true,
					},
				},
			},
		},
		"CVE-2020-8551": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.15.0",
						VersionEndIncluding:   "1.15.9",
						Vulnerable:            true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.16.0",
						VersionEndIncluding:   "1.16.6",
						Vulnerable:            true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.17.0",
						VersionEndIncluding:   "1.17.2",
						Vulnerable:            true,
					},
				},
			},
		},
		"CVE-2020-8552": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionEndExcluding: "1.15.9",
						Vulnerable:          true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.16.0",
						VersionEndIncluding:   "1.16.6",
						Vulnerable:            true,
					},
					{
						Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionEndExcluding: "1.17.0",
						VersionEndIncluding: "1.17.2",
						Vulnerable:          true,
					},
				},
			},
		},
		// https://github.com/kubernetes/kubernetes/issues/91507
		"CVE-2020-10749": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionEndExcluding: "1.16.11",
						Vulnerable:          true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.17.0",
						VersionEndIncluding:   "1.17.6",
						Vulnerable:            true,
					},
					{
						Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionEndExcluding: "1.18.0",
						VersionEndIncluding: "1.18.3",
						Vulnerable:          true,
					},
				},
			},
		},
		// https://github.com/kubernetes/kubernetes/issues/93032
		"CVE-2020-8557": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionEndExcluding: "1.16.13",
						Vulnerable:          true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.17.0",
						VersionEndIncluding:   "1.17.8",
						Vulnerable:            true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.18.0",
						VersionEndIncluding:   "1.18.5",
						Vulnerable:            true,
					},
				},
			},
		},
		// https://github.com/kubernetes/kubernetes/issues/92914
		"CVE-2020-8559": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionEndExcluding: "1.15.9",
						Vulnerable:          true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.16.0",
						VersionEndIncluding:   "1.16.12",
						Vulnerable:            true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.17.0",
						VersionEndIncluding:   "1.17.8",
						Vulnerable:            true,
					},
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.18.0",
						VersionEndIncluding:   "1.18.5",
						Vulnerable:            true,
					},
				},
			},
		},
		"CVE-2020-8554": {
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
						VersionStartIncluding: "1.0.0",
						Vulnerable:            true,
					},
				},
			},
		},
	}
)
