package mitre

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	data, err := os.ReadFile("testdata/mitre.json")
	assert.NoError(t, err)

	var rawBundle mitreBundle
	err = json.Unmarshal(data, &rawBundle)
	assert.NoError(t, err)

	expectedContainerMatrix := &storage.MitreAttackMatrix{
		MatrixInfo: &storage.MitreAttackMatrix_MatrixInfo{
			Domain:   Enterprise.String(),
			Platform: Container.String(),
		},
		Vectors: []*storage.MitreAttackVector{
			{
				Tactic: &storage.MitreTactic{
					Id:          "TA0006",
					Name:        "Credential Access",
					Description: "The adversary is trying to steal account names and passwords.\n\nCredential Access consists of techniques for stealing credentials like account names and passwords. Techniques used to get credentials include keylogging or credential dumping. Using legitimate credentials can give adversaries access to systems, make them harder to detect, and provide the opportunity to create more accounts to help achieve their goals.",
				},
				Techniques: []*storage.MitreTechnique{
					{
						Id:          "T1110",
						Name:        "Brute Force",
						Description: "Adversaries may use brute force techniques to gain access to accounts when passwords are unknown or when password hashes are obtained. Without knowledge of the password for an account or set of accounts, an adversary may systematically guess the password using a repetitive or iterative mechanism. Brute forcing passwords can take place via interaction with a service that will check the validity of those credentials or offline against previously acquired credential data, such as password hashes.",
					},
				},
			},
		},
	}

	expectedLinuxMatrix := &storage.MitreAttackMatrix{
		MatrixInfo: &storage.MitreAttackMatrix_MatrixInfo{
			Domain:   Enterprise.String(),
			Platform: Linux.String(),
		},
		Vectors: []*storage.MitreAttackVector{
			{
				Tactic: &storage.MitreTactic{
					Id:          "TA0006",
					Name:        "Credential Access",
					Description: "The adversary is trying to steal account names and passwords.\n\nCredential Access consists of techniques for stealing credentials like account names and passwords. Techniques used to get credentials include keylogging or credential dumping. Using legitimate credentials can give adversaries access to systems, make them harder to detect, and provide the opportunity to create more accounts to help achieve their goals.",
				},
				Techniques: []*storage.MitreTechnique{
					{
						Id:          "T1110",
						Name:        "Brute Force",
						Description: "Adversaries may use brute force techniques to gain access to accounts when passwords are unknown or when password hashes are obtained. Without knowledge of the password for an account or set of accounts, an adversary may systematically guess the password using a repetitive or iterative mechanism. Brute forcing passwords can take place via interaction with a service that will check the validity of those credentials or offline against previously acquired credential data, such as password hashes.",
					},
					{
						Id:          "T1110.002",
						Name:        "Brute Force: Password Cracking",
						Description: "Adversaries may use password cracking to attempt to recover usable credentials, such as plaintext passwords, when credential material such as password hashes are obtained. [OS Credential Dumping](https://attack.mitre.org/techniques/T1003) is used to obtain password hashes, this may only get an adversary so far when [Pass the Hash](https://attack.mitre.org/techniques/T1550/002) is not an option. Techniques to systematically guess the passwords used to compute hashes are available, or the adversary may use a pre-computed rainbow table to crack hashes. Cracking hashes is usually done on adversary-controlled systems outside of the target network.(Citation: Wikipedia Password cracking) The resulting plaintext password resulting from a successfully cracked hash may be used to log into systems, resources, and services in which the account has access.",
					},
				},
			},
		},
	}

	bundles, err := ExtractMitreAttackBundle(Enterprise, []Platform{Container}, rawBundle.Objects)
	assert.NoError(t, err)
	assert.Equal(t, &storage.MitreAttackBundle{
		Version: "9.0",
		Matrices: []*storage.MitreAttackMatrix{
			expectedContainerMatrix,
		},
	}, bundles)

	bundles, err = ExtractMitreAttackBundle(Enterprise, []Platform{Linux, Container}, rawBundle.Objects)
	assert.NoError(t, err)
	assert.Equal(t, &storage.MitreAttackBundle{
		Version: "9.0",
		Matrices: []*storage.MitreAttackMatrix{
			expectedContainerMatrix,
			expectedLinuxMatrix,
		},
	}, bundles)
}
