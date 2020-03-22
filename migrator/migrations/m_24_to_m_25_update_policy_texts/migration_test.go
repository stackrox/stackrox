package m24to25

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

type MigrationTestSuite struct {
	suite.Suite
	db *bolt.DB
}

func (suite *MigrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(policyBucketName)
		return err
	}))
	suite.db = db
}

func (suite *MigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertThing(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *MigrationTestSuite) mustInsertPolicy(p *storage.Policy) {
	policyBucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	suite.NoError(insertThing(policyBucket, p.GetId(), p))
}

func (suite *MigrationTestSuite) TestUpdatePolicyTexts() {
	oldPolicies := []*storage.Policy{
		//Fixable CVSS >=6 and Privileged
		{
			Id:          "93f4b2dd-ef5a-419e-8371-38aed480fb36",
			Description: "Alert on deployments with vulnerabilities with a CVSS score >= 6 and running in Privileged Mode",
			Rationale:   "If an image has a vulnerability with a CVSS score greater than 6 and the corresponding container is running as privileged, the container may pose an unnecessarily high risk of exploitation and privilege escalation.",
			Remediation: "Speak with your security team to identify and mitigate the vulnerabilities or run your container with fewer privileges.",
		},
		//Fixable CVSS >=7
		{
			Id:          "f09f8da1-6111-4ca0-8f49-294a76c65115",
			Description: "Alert on deployments with a vulnerability with a CVSS >= 7",
			Rationale:   "If an image has a vulnerability with a CVSS greater than 7, the container may pose an unnecessarily high risk of exploitation.",
			Remediation: "Speak with your security team to identify and mitigate the vulnerabilities.",
		},
		//Compiler Tool Execution
		{
			Id:          "101952d3-ec69-4ebe-bfa3-ff26b6e4c29d",
			Description: "Detects execution of binaries which are used to compile software",
			Rationale:   "Usage of compilation tools during runtime may indicate usage of infrastructure to build software.",
			Remediation: "Avoid packaging compiler tools container image.",
		},
		//30-Day Scan Age
		{
			Id:          "a3eb6dbe-e9ca-451a-919b-216cf7ee11f5",
			Description: "Alert on deployments with images that haven't been scanned in 30 days",
			Rationale:   "Out-of-date scans may not identify the most recent CVEs.",
			Remediation: "Trigger a scan on your images, configure a schedule within the scanner, or automate periodic updates to your scans.",
		},
		//Alpine Linux Package Manager Execution
		{
			Id:          "d63564bd-c184-40bc-9f30-39711e010b82",
			Description: "Alert on deployments with the Alpine Linux package manager (apk) is executed in runtime",
			Rationale:   "Package managers make it easier for attackers to use compromised containers, since they can easily add software.",
			Remediation: "Run `apk --purge del apk-tools` in the image build for production containers.",
		},
		// Red Hat Package Manager Execution
		{
			Id:          "ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce",
			Description: "Detects execution of binaries which are components of the Red Hat/Fedora/CentOS package management system.",
			Rationale:   "Package managers make it easier for attackers to use compromised containers, since they can easily add software.",
			Remediation: "Run `rpm -e $(rpm -qa *rpm*) $(rpm -qa *dnf*) $(rpm -qa *libsolv*) $(rpm -qa *hawkey*)` in the image build for production containers.",
		},
		//Ubuntu Package Manager Execution
		{
			Id:          "d7a275e1-1bba-47e7-92a1-42340c759883",
			Description: "Detects execution of binaries which are components of the Debian/Ubuntu package management system.",
			Rationale:   "Package managers make it easier for attackers to use compromised containers, since they can easily add software.",
			Remediation: "Run `apt-get remove -y --allow-remove-essential apt` in the image build for production containers.",
		},
		// Red Hat Package Manager in Image
		{
			Id:          "f95ff08d-130a-465a-a27e-32ed1fb05555",
			Description: "Alert on deployments with compoenents of the Red Hat/Fedora/CentOS package management system.",
			Rationale:   "Package managers make it easier for attackers to use compromised containers, since they can easily add software.",
			Remediation: "Run `rpm -e $(rpm -qa *rpm*) $(rpm -qa *dnf*) $(rpm -qa *libsolv*) $(rpm -qa *hawkey*) $(rpm -qa yum*)` in the image build for production containers.",
		},
	}

	expectedPolicies := []*storage.Policy{
		//Fixable CVSS >=6 and Privileged
		{
			Id:          "93f4b2dd-ef5a-419e-8371-38aed480fb36",
			Description: "Alert on deployments running in privileged mode with fixable vulnerabilities with a CVSS of at least 6",
			Rationale:   "Known vulnerabilities make it easier for adversaries to exploit your application, and highly privileged containers pose greater risk. You can fix these high-severity vulnerabilities by updating to a newer version of the affected component(s).",
			Remediation: "Use your package manager to update to a fixed version in future builds, run your container with lower privileges, or speak with your security team to mitigate the vulnerabilities.",
		},
		//Fixable CVSS >=7
		{
			Id:          "f09f8da1-6111-4ca0-8f49-294a76c65115",
			Description: "Alert on deployments with fixable vulnerabilities with a CVSS of at least 7",
			Rationale:   "Known vulnerabilities make it easier for adversaries to exploit your application. You can fix these high-severity vulnerabilities by updating to a newer version of the affected component(s).",
			Remediation: "Use your package manager to update to a fixed version in future builds or speak with your security team to mitigate the vulnerabilities.",
		},
		//Compiler Tool Execution
		{
			Id:          "101952d3-ec69-4ebe-bfa3-ff26b6e4c29d",
			Description: "Alert when binaries used to compile software are executed at runtime",
			Rationale:   "Use of compilation tools during runtime indicates that new software may be being introduced into containers while they are running.",
			Remediation: "Compile all necessary application code during the image build process. Avoid packaging software build tools in container images. Use your distribution's package manager to remove compilers and other build tools from images.",
		},
		//30-Day Scan Age
		{
			Id:          "a3eb6dbe-e9ca-451a-919b-216cf7ee11f5",
			Description: "Alert on deployments with images that haven't been scanned in 30 days",
			Rationale:   "Out-of-date scans may not identify the most recent CVEs.",
			Remediation: "Integrate a scanner with the StackRox Kubernetes Security Platform to trigger scans automatically.",
		},

		//Alpine Linux Package Manager Execution
		{
			Id:          "d63564bd-c184-40bc-9f30-39711e010b82",
			Description: "Alert when the Alpine Linux package manager (apk) is executed at runtime",
			Rationale:   "Use of package managers at runtime indicates that new software may be being introduced into containers while they are running.",
			Remediation: "Run `apk --purge del apk-tools` in the image build for production containers. Change applications to no longer use package managers at runtime, if applicable.",
		},
		//Red Hat Package Manager Execution
		{
			Id:          "ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce",
			Description: "Alert when Red Hat/Fedora/CentOS package manager programs are executed at runtime",
			Rationale:   "Use of package managers at runtime indicates that new software may be being introduced into containers while they are running.",
			Remediation: "Run `rpm -e $(rpm -qa *rpm*) $(rpm -qa *dnf*) $(rpm -qa *libsolv*) $(rpm -qa *hawkey*)` in the image build for production containers. Change applications to no longer use package managers at runtime, if applicable.",
		},
		//Ubuntu Package Manager Execution
		{
			Id:          "d7a275e1-1bba-47e7-92a1-42340c759883",
			Description: "Alert when Debian/Ubuntu package manager programs are executed at runtime",
			Rationale:   "Use of package managers at runtime indicates that new software may be being introduced into containers while they are running.",
			Remediation: "Run `apt-get remove -y --allow-remove-essential apt` in the image build for production containers. Change applications to no longer use package managers at runtime, if applicable.",
		},
		// Red Hat Package Manager in Image
		{
			Id:          "f95ff08d-130a-465a-a27e-32ed1fb05555",
			Description: "Alert on deployments with components of the Red Hat/Fedora/CentOS package management system.",
			Rationale:   "Package managers make it easier for attackers to use compromised containers, since they can easily add software.",
			Remediation: "Run `rpm -e $(rpm -qa *rpm*) $(rpm -qa *dnf*) $(rpm -qa *libsolv*) $(rpm -qa *hawkey*) $(rpm -qa yum*)` in the image build for production containers.",
		},
	}

	for _, p := range oldPolicies {
		suite.mustInsertPolicy(p)
	}

	suite.NoError(migration.Run(suite.db, nil))

	actualPolicies := make([]*storage.Policy, 0, len(oldPolicies))
	policyBucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	suite.NoError(policyBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			var policyValue storage.Policy
			err := proto.Unmarshal(v, &policyValue)
			if err != nil {
				return err
			}
			actualPolicies = append(actualPolicies, &policyValue)
			return nil
		})
	}))
	suite.ElementsMatch(expectedPolicies, actualPolicies)
}
