package metadata

var hipaa164 = Standard{
	ID:   "HIPAA_164",
	Name: "HIPAA 164",
	Categories: []Category{
		{
			ID:          "306",
			Name:        "306",
			Description: "General rules",
			Controls: []Control{
				{
					ID:          "306_e",
					Name:        "306.e",
					Description: "Maintenance of Health related documents",
				},
			},
		},
		{
			ID:          "308",
			Name:        "308",
			Description: "Administrative safeguards",
			Controls: []Control{
				{
					ID:          "308_a_1_i",
					Name:        "308.a.1.i",
					Description: "Security Management Process",
				},
				{
					ID:          "308_a_1_ii_a",
					Name:        "308.a.1.ii.a",
					Description: "Security Awareness and Training",
				},
				{
					ID:          "308_a_1_ii_b",
					Name:        "308.a.1.ii.b",
					Description: "Security Management Process",
				},
				{
					ID:          "308_a_3_ii_a",
					Name:        "308.a.3.ii.a",
					Description: `Implement procedures for the authorization and/or supervision of workforce members who work with electronic protected health information or in locations where it might be accessed`,
				},
				{
					ID:          "308_a_3_ii_b",
					Name:        "308.a.3.ii.b",
					Description: `Implement procedures to determine that the access of a workforce member to electronic protected health information is appropriate.`,
				},
				{
					ID:          "308_a_4",
					Name:        "308.a.4",
					Description: "Information Access Management",
				},
				{
					ID:          "308_a_4_ii_b",
					Name:        "308.a.4.ii.b",
					Description: "Information Access Management",
				},
				{
					ID:          "308_a_5_ii_b",
					Name:        "308.a.5.ii.b",
					Description: "Security Awareness and Training",
				},
				{
					ID:          "308_a_6_ii",
					Name:        "308.a.6.ii",
					Description: "Identify and respond to suspected or known security incidents",
				},
				{
					ID:          "308_a_7_ii_e",
					Name:        "308.a.7.ii.e",
					Description: "Applications and data criticality analysis",
				},
			},
		},
		{
			ID:          "310",
			Name:        "310",
			Description: "Physical safeguards",
			Controls: []Control{
				{
					ID:          "310_a_1",
					Name:        "310.a.1",
					Description: `Implement policies and procedures to limit physical access to its electronic information systems and the facility or facilities in which they are housed, while ensuring that properly authorized access is allowed`,
				},
				{
					ID:          "310_d",
					Name:        "310.d",
					Description: "Device and media controls",
				},
			},
		},
		{
			ID:          "312",
			Name:        "312",
			Description: "Technical safeguards",
			Controls: []Control{
				{
					ID:   "312_c",
					Name: "312.c",
					Description: `Implement policies and procedures to protect electronic protected health information 
					from improper alteration or destruction.`,
				},
				{
					ID:          "312_e",
					Name:        "312.e",
					Description: "Implement technical security measures to guard against unauthorized access to electronic protected health information that is being transmitted over an electronic communications network.",
				},
				{
					ID:          "312_e_1",
					Name:        "312.e.1",
					Description: "Implement technical security measures to guard against unauthorized access to electronic protected health information that is being transmitted over an electronic communications network.",
				},
			},
		},
		{
			ID:          "314",
			Name:        "314",
			Description: `Organizational requirements`,
			Controls: []Control{
				{
					ID:   "314_a_2_i_c",
					Name: "314.a.2.i.c",
					Description: `Report to the covered entity any security incident of which it becomes aware, 
					including breaches of unsecured protected health information.`,
				},
			},
		},
		{
			ID:          "316",
			Name:        "316",
			Description: "Policies and procedures",
			Controls: []Control{
				{
					ID:          "316_b_2_iii",
					Name:        "316.b.2.iii",
					Description: "Review documentation periodically, and update as needed, in response to environmental or operational changes affecting the security of the electronic protected health information.",
				},
			},
		},
	},
}

func init() {
	AllStandards = append(AllStandards, hipaa164)
}
