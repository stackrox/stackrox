package common

import (
	"github.com/stackrox/rox/generated/storage"
)

// This file will be deleted once the actual data is available. It is added only to enable UI work.

func init() {
	registerSampleData()
}

func registerSampleData() {
	store := rwSingleton()
	store.add(
		"TA0006", &storage.MitreAttackVector{
			Tactic: &storage.MitreTactic{
				Id:   "TA0006",
				Name: "Credential Access",
				Description: "The adversary is trying to steal account names and passwords. Credential Access " +
					"consists of techniques for stealing credentials like account names and passwords. " +
					"Techniques used to get credentials include keylogging or credential dumping. Using " +
					"legitimate credentials can give adversaries access to systems, make them harder to detect, " +
					"and provide the opportunity to create more accounts to help achieve their goals.",
			},
			Techniques: []*storage.MitreTechnique{
				{
					Id:   "T1110",
					Name: "Brute Force",
					Description: "Adversaries may use brute force techniques to gain access to accounts when " +
						"passwords are unknown or when password hashes are obtained. Without knowledge of the " +
						"password for an account or set of accounts, an adversary may systematically guess the " +
						"password using a repetitive or iterative mechanism. Brute forcing passwords can take " +
						"place via interaction with a service that will check the validity of those credentials " +
						"or offline against previously acquired credential data, such as password hashes.",
				},
				{
					Id:   "T1552",
					Name: "Unsecured Credentials",
					Description: "Adversaries may search compromised systems to find and obtain insecurely " +
						"stored credentials. These credentials can be stored and/or misplaced in many locations " +
						"on a system, including plaintext files (e.g. Bash History), operating system or " +
						"application-specific repositories (e.g. Credentials in Registry), or other specialized " +
						"files/artifacts (e.g. Private Keys).",
				},
			},
		},
	)
}
