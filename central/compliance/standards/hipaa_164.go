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
	},
}

func init() {
	utils.Must(RegistrySingleton().RegisterStandard(&hipaa164))
}
