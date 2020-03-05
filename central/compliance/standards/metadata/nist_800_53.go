package metadata

import (
	"github.com/stackrox/rox/pkg/features"
)

var nist800_53 = Standard{
	ID:   "NIST_SP_800_53_Rev_4",
	Name: "NIST SP 800-53",
	Categories: []Category{
		{
			ID:          "AC",
			Name:        "AC",
			Description: "Access Control",
			Controls: []Control{
				{
					ID:          "AC_14",
					Name:        "AC-14",
					Description: "Permitted Actions Without Identification Or Authentication",
				},
				{
					ID:          "AC_24",
					Name:        "AC-24",
					Description: "Access Control Decisions",
				},
				{
					ID:          "AC_3_(7)",
					Name:        "AC-3 (7)",
					Description: "Role-Based Access Control",
				},
			},
		},
		{
			ID:          "CA",
			Name:        "CA",
			Description: "Security Assessment And Authorization",
			Controls: []Control{
				{
					ID:          "CA_9",
					Name:        "CA-9",
					Description: "Internal System Connections",
				},
			},
		},
		{
			ID:          "CM",
			Name:        "CM",
			Description: "Configuration Management",
			Controls: []Control{
				{
					ID:          "CM_11",
					Name:        "CM-11",
					Description: "User-Installed Software",
				},
				{
					ID:          "CM_2",
					Name:        "CM-2",
					Description: "Baseline Configuration",
				},
				{
					ID:          "CM_3",
					Name:        "CM-3",
					Description: "Configuration Change Control",
				},
				{
					ID:          "CM_5",
					Name:        "CM-5",
					Description: "Access Restrictions For Change",
				},
				{
					ID:          "CM_6",
					Name:        "CM-6",
					Description: "Configuration Settings",
				},
				{
					ID:          "CM_7",
					Name:        "CM-7",
					Description: "Least Functionality",
				},
				{
					ID:          "CM_8",
					Name:        "CM-8",
					Description: "Information System Component Inventory",
				},
			},
		},
		{
			ID:          "IR",
			Name:        "IR",
			Description: "Incident Response",
			Controls: []Control{
				{
					ID:          "IR_4_(5)",
					Name:        "IR-4 (5)",
					Description: "Automatic Disabling Of Information System",
				},
				{
					ID:          "IR_5",
					Name:        "IR-5",
					Description: "Incident Monitoring",
				},
				{
					ID:          "IR_6_(1)",
					Name:        "IR-6 (1)",
					Description: "Automated Reporting",
				},
			},
		},
		{
			ID:          "RA",
			Name:        "RA",
			Description: "Risk Assessment",
			Controls: []Control{
				{
					ID:          "RA_3",
					Name:        "RA-3",
					Description: "Risk Assessment",
				},
				{
					ID:          "RA_5",
					Name:        "RA-5",
					Description: "Vulnerability Scanning",
				},
			},
		},
		{
			ID:          "SA",
			Name:        "SA",
			Description: "System and Services Acquisition",
			Controls: []Control{
				{
					ID:          "SA_10",
					Name:        "SA-10",
					Description: "Developer Configuration Management",
				},
			},
		},
		{
			ID:          "SC",
			Name:        "SC",
			Description: "System And Communications Protection",
			Controls: []Control{
				{
					ID:          "SC_6",
					Name:        "SC-6",
					Description: "Resource Availability",
				},
				{
					ID:          "SC_7",
					Name:        "SC-7",
					Description: "Boundary Protection",
				},
			},
		},
		{
			ID:          "SI",
			Name:        "SI",
			Description: "System And Information Integrity",
			Controls: []Control{
				{
					ID:          "SI_2_(2)",
					Name:        "SI-2 (2)",
					Description: "Automated Flaw Remediation Status",
				},
				{
					ID:          "SI_3_(8)",
					Name:        "SI-3 (8)",
					Description: "Detect Unauthorized Commands",
				},
				{
					ID:          "SI_4",
					Name:        "SI-4",
					Description: "Information System Monitoring",
				},
			},
		},
	},
}

func init() {
	if features.NistSP800_53.Enabled() {
		AllStandards = append(AllStandards, nist800_53)
	}
}
