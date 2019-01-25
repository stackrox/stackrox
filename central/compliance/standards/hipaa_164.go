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
	},
}

func init() {
	utils.Must(RegistrySingleton().RegisterStandard(&hipaa164))
}
