package datastore

import (
	"github.com/stackrox/rox/generated/storage"
)

// TODO(@Mandar): ROX-7749: Remove sample data when feature is turned on by default

var (
	// MitreTestData is test data to enable UI work.
	MitreTestData = map[string]*storage.MitreAttackVector{
		"TA0006": {
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
		"TA0005": {
			Tactic: &storage.MitreTactic{
				Id:   "TA0005",
				Name: "Defense Evasion",
				Description: "Defense Evasion consists of techniques that adversaries use to avoid detection" +
					" throughout their compromise. Techniques used for defense evasion include " +
					"uninstalling/disabling security software or obfuscating/encrypting data and scripts. " +
					"Adversaries also leverage and abuse trusted processes to hide and masquerade their malware. " +
					"Other tactics’ techniques are cross-listed here when those techniques include the added " +
					"benefit of subverting defenses.",
			},
			Techniques: []*storage.MitreTechnique{
				{
					Id:   "T1562",
					Name: "Impair Defenses",
					Description: "Adversaries may maliciously modify components of a victim environment in " +
						"order to hinder or disable defensive mechanisms. This not only involves impairing " +
						"preventative defenses, such as firewalls and anti-virus, but also detection capabilities " +
						"that defenders can use to audit activity and identify malicious behavior. This may also " +
						"span both native defenses as well as supplemental capabilities installed by users and administrators.",
				},
				{
					Id:   "T1610",
					Name: "Deploy Container",
					Description: "Adversaries may deploy a container into an environment to facilitate execution " +
						"or evade defenses. In some cases, adversaries may deploy a new container to execute " +
						"processes associated with a particular image or deployment, such as processes that " +
						"execute or download malware. In others, an adversary may deploy a new container configured" +
						" without network rules, user limitations, etc. to bypass existing defenses within the environment.",
				},
			},
		},
	}
)

func init() {
	registerSampleData()
}

func registerSampleData() {
	store := rwSingleton()
	store.add("TA0006", MitreTestData["TA0006"])
	store.add("TA0005", MitreTestData["TA0005"])
}
