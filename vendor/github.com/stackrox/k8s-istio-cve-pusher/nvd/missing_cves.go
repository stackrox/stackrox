package nvd

import "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"

var (
	// cvesNotInDataFeed represents NVD CVEs that may not be available in NVD data feeds
	cvesNotInDataFeed = map[Project]map[string]*schema.NVDCVEFeedJSON10DefCVEItem{
		Kubernetes: {
			"CVE-2020-8551": {
				PublishedDate:    "2020-03-27T15:15Z",
				LastModifiedDate: "2020-03-27T16:03Z",
				CVE: &schema.CVEJSON40{
					CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
						ID: "CVE-2020-8551",
					},
					Description: &schema.CVEJSON40Description{
						DescriptionData: []*schema.CVEJSON40LangString{
							{
								Value: "The Kubelet component in versions 1.15.0-1.15.9, 1.16.0-1.16.6, and 1.17.0-1.17.2 has been found to be vulnerable to a denial of service attack via the kubelet API, including the unauthenticated HTTP read-only API typically served on port 10255, and the authenticated HTTPS API typically served on port 10250.",
							},
						},
					},
				},
				Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
					Nodes: []*schema.NVDCVEFeedJSON10DefNode{
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
				},
			},
			"CVE-2020-8552": {
				PublishedDate:    "2020-03-27T15:15Z",
				LastModifiedDate: "2020-03-27T16:03Z",
				CVE: &schema.CVEJSON40{
					CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
						ID: "CVE-2020-8552",
					},
					Description: &schema.CVEJSON40Description{
						DescriptionData: []*schema.CVEJSON40LangString{
							{
								Value: "The Kubernetes API server component in versions prior to 1.15.9, 1.16.0-1.16.6, and 1.17.0-1.17.2 has been found to be vulnerable to a denial of service attack via successful API requests.",
							},
						},
					},
				},
				Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
					Nodes: []*schema.NVDCVEFeedJSON10DefNode{
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
				},
			},
			"CVE-2020-10749": {
				CVE: &schema.CVEJSON40{
					CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
						ID: "CVE-2020-10749",
					},
					Description: &schema.CVEJSON40Description{
						DescriptionData: []*schema.CVEJSON40LangString{
							{
								Value: "The Kubelet component in versions prior to v1.16.11, v1.17.0-v1.17.6, and v1.18.0-v1.18.3 have an affected kubernetes-cni package that has been found vulnerable to man-in-the-middle attacks.",
							},
						},
					},
				},
				Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
					Nodes: []*schema.NVDCVEFeedJSON10DefNode{},
				},
			},
			"CVE-2020-8557": {
				CVE: &schema.CVEJSON40{
					CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
						ID: "CVE-2020-8557",
					},
					Description: &schema.CVEJSON40Description{
						DescriptionData: []*schema.CVEJSON40LangString{
							{
								Value: "The /etc/hosts file mounted in a pod by kubelet is not included by the kubelet eviction manager when calculating ephemeral storage usage by a pod. If a pod writes a large amount of data to the /etc/hosts file, it could fill the storage space of the node and cause the node to fail. This affects kublet v1.18.0-1.18.5, kubelet v1.17.0-1.17.8, and kubelet < v1.16.13.",
							},
						},
					},
				},
				Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
					Nodes: []*schema.NVDCVEFeedJSON10DefNode{
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
				},
			},
			"CVE-2020-8559": {
				CVE: &schema.CVEJSON40{
					CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
						ID: "CVE-2020-8559",
					},
					Description: &schema.CVEJSON40Description{
						DescriptionData: []*schema.CVEJSON40LangString{
							{
								Value: "If an attacker is able to intercept certain requests to the Kubelet, they can send a redirect response that may be followed by a client using the credentials from the original request. This can lead to compromise of other nodes. If multiple clusters share the same certificate authority trusted by the client, and the same authentication credentials, this vulnerability may allow an attacker to redirect the client to another cluster. In this configuration, this vulnerability should be considered High severity. This affects kube-apiserver v1.18.0-1.18.5, kube-apiserver v1.17.0-1.17.8, and all kube-apiserver versions prior to v1.16.0.",
							},
						},
					},
				},
				Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
					Nodes: []*schema.NVDCVEFeedJSON10DefNode{
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
				},
			},
			"CVE-2020-8554": {
				CVE: &schema.CVEJSON40{
					CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
						ID: "CVE-2020-8554",
					},
					Description: &schema.CVEJSON40Description{
						DescriptionData: []*schema.CVEJSON40LangString{
							{
								Value: "This issue affects multitenant clusters. If a potential attacker can already create or edit services and pods, then they may be able to intercept traffic from other pods (or nodes) in the cluster. An attacker that is able to create a ClusterIP service and set the spec.externalIPs field can intercept traffic to that IP. An attacker that is able to patch the status (which is considered a privileged operation and should not typically be granted to users) of a LoadBalancer service can set the status.loadBalancer.ingress.ip to similar effect. This issue is a design flaw that cannot be mitigated without user-facing changes.",
							},
						},
					},
				},
				Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
					Nodes: []*schema.NVDCVEFeedJSON10DefNode{
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
				},
				PublishedDate:    "2020-12-09T17:09Z",
				LastModifiedDate: "2020-12-09T17:09Z",
				Impact: &schema.NVDCVEFeedJSON10DefImpact{
					BaseMetricV3: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
						CVSSV3: &schema.CVSSV30{
							AttackComplexity:      "LOW",
							AttackVector:          "NETWORK",
							AvailabilityImpact:    "LOW",
							BaseScore:             6.3,
							BaseSeverity:          "MEDIUM",
							ConfidentialityImpact: "LOW",
							IntegrityImpact:       "LOW",
							PrivilegesRequired:    "LOW",
							Scope:                 "UNCHANGED",
							UserInteraction:       "NONE",
							VectorString:          "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L",
							Version:               "3.0",
						},
						ExploitabilityScore: 2.8,
						ImpactScore:         3.4,
					},
					BaseMetricV2: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV2{
						CVSSV2: &schema.CVSSV20{
							AccessComplexity:      "LOW",
							AccessVector:          "NETWORK",
							Authentication:        "SINGLE",
							AvailabilityImpact:    "PARTIAL",
							BaseScore:             6.5,
							ConfidentialityImpact: "PARTIAL",
							IntegrityImpact:       "PARTIAL",
							VectorString:          "AV:N/AC:L/Au:S/C:P/I:P/A:P",
							Version:               "2.0",
						},
						ExploitabilityScore: 8.0,
						ImpactScore:         6.4,
						Severity:            "MEDIUM",
					},
				},
			},
			"CVE-2021-25735": {
				CVE: &schema.CVEJSON40{
					CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
						ID: "CVE-2021-25735",
					},
					Description: &schema.CVEJSON40Description{
						DescriptionData: []*schema.CVEJSON40LangString{
							{
								Value: "A security issue was discovered in kube-apiserver that could allow node updates to bypass a Validating Admission Webhook. You are only affected by this vulnerability if you run a Validating Admission Webhook for Nodes that denies admission based at least partially on the old state of the Node object.",
							},
						},
					},
				},
				Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
					Nodes: []*schema.NVDCVEFeedJSON10DefNode{
						{
							Operator: "OR",
							CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
								{
									Cpe23Uri:            "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
									VersionEndExcluding: "1.18.18",
									Vulnerable:          true,
								},
								{
									Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
									VersionStartIncluding: "1.19.0",
									VersionEndExcluding:   "1.19.10",
									Vulnerable:            true,
								},
								{
									Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
									VersionStartIncluding: "1.20.0",
									VersionEndExcluding:   "1.20.6",
									Vulnerable:            true,
								},
							},
						},
					},
				},
				PublishedDate:    "2021-04-14T10:35Z",
				LastModifiedDate: "2021-04-14T10:35Z",
				Impact: &schema.NVDCVEFeedJSON10DefImpact{
					BaseMetricV3: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
						CVSSV3: &schema.CVSSV30{
							AttackComplexity:      "LOW",
							AttackVector:          "NETWORK",
							AvailabilityImpact:    "HIGH",
							BaseScore:             6.5,
							BaseSeverity:          "MEDIUM",
							ConfidentialityImpact: "NONE",
							IntegrityImpact:       "HIGH",
							PrivilegesRequired:    "HIGH",
							Scope:                 "UNCHANGED",
							UserInteraction:       "NONE",
							VectorString:          "CVSS:3.0/AV:N/AC:L/PR:H/UI:N/S:U/C:N/I:H/A:H",
							Version:               "3.0",
						},
						ExploitabilityScore: 1.2,
						ImpactScore:         5.2,
					},
					BaseMetricV2: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV2{
						CVSSV2: &schema.CVSSV20{
							AccessComplexity:      "LOW",
							AccessVector:          "NETWORK",
							Authentication:        "SINGLE",
							AvailabilityImpact:    "PARTIAL",
							BaseScore:             6.5,
							ConfidentialityImpact: "PARTIAL",
							IntegrityImpact:       "PARTIAL",
							VectorString:          "AV:N/AC:L/Au:S/C:P/I:P/A:P",
							Version:               "2.0",
						},
						ExploitabilityScore: 8.0,
						ImpactScore:         6.4,
						Severity:            "MEDIUM",
					},
				},
			},
		},
	}
)
