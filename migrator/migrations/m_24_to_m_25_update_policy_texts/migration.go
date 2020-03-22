package m24to25

import (
	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

type policyMeta struct {
	id          string
	description string
	rationale   string
	remediation string
}

var (
	policyBucketName = []byte("policies")

	cvssGreaterThan6Policy = policyMeta{
		id:          "93f4b2dd-ef5a-419e-8371-38aed480fb36",
		description: "Alert on deployments running in privileged mode with fixable vulnerabilities with a CVSS of at least 6",
		rationale:   "Known vulnerabilities make it easier for adversaries to exploit your application, and highly privileged containers pose greater risk. You can fix these high-severity vulnerabilities by updating to a newer version of the affected component(s).",
		remediation: "Use your package manager to update to a fixed version in future builds, run your container with lower privileges, or speak with your security team to mitigate the vulnerabilities.",
	}

	cvssGreaterThan7Policy = policyMeta{
		id:          "f09f8da1-6111-4ca0-8f49-294a76c65115",
		description: "Alert on deployments with fixable vulnerabilities with a CVSS of at least 7",
		rationale:   "Known vulnerabilities make it easier for adversaries to exploit your application. You can fix these high-severity vulnerabilities by updating to a newer version of the affected component(s).",
		remediation: "Use your package manager to update to a fixed version in future builds or speak with your security team to mitigate the vulnerabilities.",
	}

	compilerToolExecutionPolicy = policyMeta{
		id:          "101952d3-ec69-4ebe-bfa3-ff26b6e4c29d",
		description: "Alert when binaries used to compile software are executed at runtime",
		rationale:   "Use of compilation tools during runtime indicates that new software may be being introduced into containers while they are running.",
		remediation: "Compile all necessary application code during the image build process. Avoid packaging software build tools in container images. Use your distribution's package manager to remove compilers and other build tools from images.",
	}

	thirtyDayScanAgePolicy = policyMeta{
		id:          "a3eb6dbe-e9ca-451a-919b-216cf7ee11f5",
		description: "Alert on deployments with images that haven't been scanned in 30 days",
		rationale:   "Out-of-date scans may not identify the most recent CVEs.",
		remediation: "Integrate a scanner with the StackRox Kubernetes Security Platform to trigger scans automatically.",
	}

	alpineLinuxPolicy = policyMeta{
		id:          "d63564bd-c184-40bc-9f30-39711e010b82",
		description: "Alert when the Alpine Linux package manager (apk) is executed at runtime",
		rationale:   "Use of package managers at runtime indicates that new software may be being introduced into containers while they are running.",
		remediation: "Run `apk --purge del apk-tools` in the image build for production containers. Change applications to no longer use package managers at runtime, if applicable.",
	}

	redHatPolicy = policyMeta{
		id:          "ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce",
		description: "Alert when Red Hat/Fedora/CentOS package manager programs are executed at runtime",
		rationale:   "Use of package managers at runtime indicates that new software may be being introduced into containers while they are running.",
		remediation: "Run `rpm -e $(rpm -qa *rpm*) $(rpm -qa *dnf*) $(rpm -qa *libsolv*) $(rpm -qa *hawkey*)` in the image build for production containers. Change applications to no longer use package managers at runtime, if applicable.",
	}

	ubuntuPolicy = policyMeta{
		id:          "d7a275e1-1bba-47e7-92a1-42340c759883",
		description: "Alert when Debian/Ubuntu package manager programs are executed at runtime",
		rationale:   "Use of package managers at runtime indicates that new software may be being introduced into containers while they are running.",
		remediation: "Run `apt-get remove -y --allow-remove-essential apt` in the image build for production containers. Change applications to no longer use package managers at runtime, if applicable.",
	}

	redHatPackageManagerinImagePolicy = policyMeta{
		id:          "f95ff08d-130a-465a-a27e-32ed1fb05555",
		description: "Alert on deployments with components of the Red Hat/Fedora/CentOS package management system.",
		rationale:   "Package managers make it easier for attackers to use compromised containers, since they can easily add software.",
		remediation: "Run `rpm -e $(rpm -qa *rpm*) $(rpm -qa *dnf*) $(rpm -qa *libsolv*) $(rpm -qa *hawkey*) $(rpm -qa yum*)` in the image build for production containers.",
	}

	policyMap = map[string]*policyMeta{
		"93f4b2dd-ef5a-419e-8371-38aed480fb36": &cvssGreaterThan6Policy,
		"f09f8da1-6111-4ca0-8f49-294a76c65115": &cvssGreaterThan7Policy,
		"101952d3-ec69-4ebe-bfa3-ff26b6e4c29d": &compilerToolExecutionPolicy,
		"a3eb6dbe-e9ca-451a-919b-216cf7ee11f5": &thirtyDayScanAgePolicy,
		"d63564bd-c184-40bc-9f30-39711e010b82": &alpineLinuxPolicy,
		"ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce": &redHatPolicy,
		"d7a275e1-1bba-47e7-92a1-42340c759883": &ubuntuPolicy,
		"f95ff08d-130a-465a-a27e-32ed1fb05555": &redHatPackageManagerinImagePolicy,
	}
)

func updatePolicyTexts(db *bolt.DB) error {
	if exists, err := bolthelpers.BucketExists(db, policyBucketName); err != nil {
		return err
	} else if !exists {
		return nil
	}
	policyBucket := bolthelpers.TopLevelRef(db, policyBucketName)
	if len(policyMap) == 0 {
		return errors.New("Policy data has something wrong in the migration program.")
	}

	return policyBucket.Update(func(b *bolt.Bucket) error {
		for key, val := range policyMap {
			v := b.Get([]byte(key))
			if v == nil {
				continue
			}
			var policyValue storage.Policy
			if err := proto.Unmarshal(v, &policyValue); err != nil {
				return err
			}
			if val != nil {
				policyValue.Description = val.description
				policyValue.Rationale = val.rationale
				policyValue.Remediation = val.remediation
			}
			policyBytes, err := proto.Marshal(&policyValue)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(key), policyBytes); err != nil {
				return errors.Wrap(err, "failed to insert")
			}
		}
		return nil
	})
}

var (
	migration = types.Migration{
		StartingSeqNum: 24,
		VersionAfter:   storage.Version{SeqNum: 25},
		Run: func(db *bolt.DB, _ *badger.DB) error {
			err := updatePolicyTexts(db)
			if err != nil {
				return errors.Wrap(err, "updating policy texts")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
