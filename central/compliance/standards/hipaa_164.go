package standards

import "github.com/stackrox/rox/pkg/utils"

var hipaa164 = Standard{
	ID:   "HIPAA_164",
	Name: "HIPAA 164",
	Categories: []Category{
		{
			ID:          "306_e",
			Name:        "306.e",
			Description: "Maintenance",
			Controls: []Control{
				{
					ID:          "306_e",
					Name:        "306.e",
					Description: "Maintenance of Health related documents",
				},
			},
		},
		{
			ID:          "308_a_1_ii_b",
			Name:        "308.a.1.ii.b",
			Description: "Security Management Process",
			Controls: []Control{
				{
					ID:          "308_a_1_ii_b",
					Name:        "308.a.1.ii.b",
					Description: "Security Management Process",
				},
			},
		},
		{
			ID:          "308_a_4_ii_b",
			Name:        "308.a.4.ii.b",
			Description: "Information Access Management",
			Controls: []Control{
				{
					ID:          "308_a_4_ii_b",
					Name:        "308.a.4.ii.b",
					Description: "Information Access Management",
				},
			},
		},
		{
			ID:          "308_a_5_ii_b",
			Name:        "308.a.5.ii.b",
			Description: "Security Awareness and Training",
			Controls: []Control{
				{
					ID:          "308_a_5_ii_b",
					Name:        "308.a.5.ii.b",
					Description: "Security Awareness and Training",
				},
			},
		},
		{
			ID:          "308_a_6_ii",
			Name:        "308.a.6.ii",
			Description: "Identify and respond to suspected or known security incidents",
			Controls: []Control{
				{
					ID:   "308_a_6_ii",
					Name: "308.a.6.ii",
					Description: "Identify and respond to suspected or known security incidents; mitigate, to the " +
						"extent practicable, harmful effects of security incidents that are known to the covered " +
						"entity or business associate; and document security incidents and their outcomes.",
				},
			},
		},
		{
			ID:          "308_a_7_ii_e",
			Name:        "308.a.7.ii.e",
			Description: "Applications and data criticality analysis",
			Controls: []Control{
				{
					ID:          "308_a_7_ii_e",
					Name:        "308.a.7.ii.e",
					Description: "Applications and data criticality analysis",
				},
			},
		},
		{
			ID:          "308_a_3_ii_a",
			Name:        "308.a.3.ii.a",
			Description: "Workforce security",
			Controls: []Control{
				{
					ID:          "308_a_3_ii_a",
					Name:        "308.a.3.ii.a",
					Description: `Implement procedures for the authorization and/or supervision of workforce members who work with electronic protected health information or in locations where it might be accessed.`,
				},
			},
		},
	},
}

func init() {
	utils.Must(RegistrySingleton().RegisterStandard(&hipaa164))
}
