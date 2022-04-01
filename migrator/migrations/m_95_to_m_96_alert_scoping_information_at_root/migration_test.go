package m95tom96

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(alertScopeInfoCopyTestSuite))
}

type alertScopeInfoCopyTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (s *alertScopeInfoCopyTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.db = rocksDB
	s.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (s *alertScopeInfoCopyTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func (s *alertScopeInfoCopyTestSuite) writeAlertToStore(alert *storage.Alert) {
	writeOpts := gorocksdb.NewDefaultWriteOptions()
	value, err := proto.Marshal(alert)
	s.NoError(err)
	err = s.db.Put(writeOpts,
		rocksdbmigration.GetPrefixedKey(alertBucket, []byte(alert.GetId())),
		value)
	s.NoError(err)
}

func (s *alertScopeInfoCopyTestSuite) checkMigratedAlertScopingInformation(alertID, clusterID, clusterName, namespace, namespaceID string) {
	readOpts := gorocksdb.NewDefaultReadOptions()
	msg, exists, err := rockshelper.ReadFromRocksDB(s.db.DB, readOpts, &storage.Alert{}, alertBucket, []byte(alertID))
	s.NoError(err)
	s.True(exists)
	readAlert := msg.(*storage.Alert)
	s.Equal(clusterID, readAlert.ClusterId)
	s.Equal(clusterName, readAlert.ClusterName)
	s.Equal(namespace, readAlert.Namespace)
	s.Equal(namespaceID, readAlert.NamespaceId)
}

func (s *alertScopeInfoCopyTestSuite) TestDeploymentAlertScopingInformationCopy() {
	alert := getAlert()
	entity := alert.GetDeployment()
	s.writeAlertToStore(alert)

	err := copyAlertScopingInformationToRoot(s.databases)
	s.NoError(err)

	s.checkMigratedAlertScopingInformation(alert.GetId(), entity.GetClusterId(), entity.GetClusterName(), entity.GetNamespace(), entity.GetNamespaceId())
}

func (s *alertScopeInfoCopyTestSuite) TestResourceAlertScopingInformationCopy() {
	alert := getResourceAlert()
	entity := alert.GetResource()
	s.writeAlertToStore(alert)

	err := copyAlertScopingInformationToRoot(s.databases)
	s.NoError(err)

	s.checkMigratedAlertScopingInformation(alert.GetId(), entity.GetClusterId(), entity.GetClusterName(), entity.GetNamespace(), entity.GetNamespaceId())
}

func (s *alertScopeInfoCopyTestSuite) TestImageAlertScopingInformationCopy() {
	alert := getImageAlert()
	s.writeAlertToStore(alert)

	err := copyAlertScopingInformationToRoot(s.databases)
	s.NoError(err)

	s.checkMigratedAlertScopingInformation(alert.GetId(), "", "", "", "")
}

// Helper functions to create test objects (duplicated from pkg/fixtures)

func getAlert() *storage.Alert {
	return &storage.Alert{
		Id: "Alert1",
		Violations: []*storage.Alert_Violation{
			{
				Message: "Deployment is affected by 'CVE-2017-15804'",
			},
			{
				Message: "Deployment is affected by 'CVE-2017-15670'",
			},
			{
				Message: "This is a kube event violation",
				MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
					KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
						Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
							{Key: "pod", Value: "nginx"},
							{Key: "container", Value: "nginx"},
						},
					},
				},
			},
		},
		ProcessViolation: &storage.Alert_ProcessViolation{
			Message: "This is a process violation",
		},
		Time:   types.TimestampNow(),
		Policy: getPolicy(),
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Name:        "nginx_server",
				Id:          "s79mdvmb6dsl",
				ClusterId:   "prod cluster",
				ClusterName: "prod cluster",
				Namespace:   "stackrox",
				Labels: map[string]string{
					"com.docker.stack.namespace":    "prevent",
					"com.docker.swarm.service.name": "prevent_sensor",
					"email":                         "vv@stackrox.com",
					"owner":                         "stackrox",
				},
				Containers: []*storage.Alert_Deployment_Container{
					{
						Name:  "nginx110container",
						Image: toContainerImage(getLightweightDeploymentImage()),
					},
				},
			},
		},
	}
}

func getResourceAlert() *storage.Alert {
	return &storage.Alert{
		Id: "some-resource-alert-on-secret",
		Violations: []*storage.Alert_Violation{
			{
				Message: "Access to secret \"my-secret\" in \"cluster-id / stackrox\"",
				Type:    storage.Alert_Violation_K8S_EVENT,
				MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
					KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
						Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
							{Key: "Kubernetes API Verb", Value: "CREATE"},
							{Key: "username", Value: "test-user"},
							{Key: "user groups", Value: "groupA, groupB"},
							{Key: "resource", Value: "/api/v1/namespace/stackrox/secrets/my-secret"},
							{Key: "user agent", Value: "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f"},
							{Key: "IP address", Value: "192.168.0.1, 127.0.0.1"},
							{Key: "impersonated username", Value: "central-service-account"},
							{Key: "impersonated user groups", Value: "service-accounts, groupB"},
						},
					},
				},
			},
		},
		Time:   types.TimestampNow(),
		Policy: getAuditLogEventSourcePolicy(),
		Entity: &storage.Alert_Resource_{
			Resource: &storage.Alert_Resource{
				ResourceType: storage.Alert_Resource_SECRETS,
				Name:         "my-secret",
				ClusterId:    "cluster-id",
				ClusterName:  "prod cluster",
				Namespace:    "stackrox",
				NamespaceId:  "aaaa-bbbb-cccc-dddd",
			},
		},
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	}
}

func getImageAlert() *storage.Alert {
	imageAlert := getAlert()
	image := getImage()
	imageAlert.Entity = &storage.Alert_Image{
		Image: toContainerImage(image),
	}

	return imageAlert
}

func toContainerImage(ci *storage.Image) *storage.ContainerImage {
	return &storage.ContainerImage{
		Id:          ci.GetId(),
		Name:        ci.GetName(),
		NotPullable: ci.GetNotPullable(),
	}
}

func getImage() *storage.Image {
	numComponentsPerImage := 50
	componentsPerImage := make([]*storage.EmbeddedImageScanComponent, 0, numComponentsPerImage)
	for i := 0; i < numComponentsPerImage; i++ {
		componentsPerImage = append(componentsPerImage, &storage.EmbeddedImageScanComponent{
			Name:    "name",
			Version: "1.2.3.4",
			License: &storage.License{
				Name: "blah",
				Type: "GPL",
			},
			Vulns: getVulnsPerComponent(i),
		})
	}
	return getImageWithComponents(componentsPerImage)
}

func getVulnsPerComponent(componentIndex int) []*storage.EmbeddedVulnerability {
	numVulnsPerComponent := 5
	vulnsPerComponent := make([]*storage.EmbeddedVulnerability, 0, numVulnsPerComponent)
	for i := 0; i < numVulnsPerComponent; i++ {
		cveName := fmt.Sprintf("CVE-2014-62%d%d", componentIndex, i)
		vulnsPerComponent = append(vulnsPerComponent, &storage.EmbeddedVulnerability{
			Cve:      cveName,
			Cvss:     5,
			Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			Summary:  "GNU Bash through 4.3 processes trailing strings after function definitions in the values of environment variables, which allows remote attackers to execute arbitrary code via a crafted environment, as demonstrated by vectors involving the ForceCommand feature in OpenSSH sshd, the mod_cgi and mod_cgid modules in the Apache HTTP Server, scripts executed by unspecified DHCP clients, and other situations in which setting the environment occurs across a privilege boundary from Bash execution, aka \"ShellShock.\"  NOTE: the original fix for this issue was incorrect; CVE-2014-7169 has been assigned to cover the vulnerability that is still present after the incorrect fix.",
			Link:     fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cveName),
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "abcdefg",
			},
		})
	}
	return vulnsPerComponent
}

func getImageWithComponents(componentsPerImage []*storage.EmbeddedImageScanComponent) *storage.Image {
	author := "author"
	return &storage.Image{
		Id: "sha256:SHA2",
		Name: &storage.ImageName{
			Registry: "stackrox.io",
			Remote:   "srox/mongo",
			Tag:      "latest",
			FullName: "stackrox.io/srox/mongo:latest",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Author:  author,
				Created: types.TimestampNow(),
				Layers: []*storage.ImageLayer{
					{
						Instruction: "CMD",
						Value:       `["nginx" "-g" "daemon off;"]`,
						Author:      author,
						Created:     types.TimestampNow(),
					},
					{
						Instruction: "EXPOSE",
						Value:       "443/tcp 80/tcp",
						Author:      author,
						Created:     types.TimestampNow(),
					},
					{
						Instruction: "RUN",
						Value:       "ln -sf /dev/stdout /var/log/nginx/access.log && ln -sf /dev/stderr /var/log/nginx/error.log",
						Author:      author,
						Created:     types.TimestampNow(),
					},
					{
						Instruction: "RUN",
						Value:       `apt-key adv --keyserver hkp://pgp.mit.edu:80 --recv-keys 573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62 && echo "deb http://nginx.org/packages/debian/ jessie nginx" >> /etc/apt/sources.list && apt-get update && apt-get install --no-install-recommends --no-install-suggests -y ca-certificates nginx=${NGINX_VERSION} nginx-module-xslt nginx-module-geoip nginx-module-image-filter nginx-module-perl nginx-module-njs gettext-base && rm -rf /var/lib/apt/lists/*`,
						Author:      author,
						Created:     types.TimestampNow(),
					},
					{
						Instruction: "ENV",
						Value:       "NGINX_VERSION=1.10.3-1~jessie",
						Author:      author,
						Created:     types.TimestampNow(),
					},
					{
						Instruction: "MAINTAINER",
						Value:       author,
						Author:      author,
						Created:     types.TimestampNow(),
					},
					{
						Instruction: "CMD",
						Value:       `["/bin/bash"]`,
						Created:     types.TimestampNow(),
					},
					{
						Instruction: "ADD",
						Value:       "file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /",
						Created:     types.TimestampNow(),
					},
				},
			},
			V2: &storage.V2Metadata{
				Digest: "sha256:0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455",
			},
			LayerShas: []string{
				"sha256:6d827a3ef358f4fa21ef8251f95492e667da826653fd43641cef5a877dc03a70",
				"sha256:1e3e18a64ea9924fd9688d125c2844c4df144e41b1d2880a06423bca925b778c",
				"sha256:556c62bb43ac9073f4dfc95383e83f8048633a041cb9e7eb2c1f346ba39a5183",
			},
		},
		Scan: &storage.ImageScan{
			ScanTime:   types.TimestampNow(),
			Components: componentsPerImage,
		},
	}
}

func getLightweightDeploymentImage() *storage.Image {
	return &storage.Image{
		Id: "sha256:SHA1",
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Layers: []*storage.ImageLayer{
					{
						Instruction: "ADD",
						Value:       "FILE:blah",
					},
				},
			},
		},
		Scan: &storage.ImageScan{
			ScanTime: types.TimestampNow(),
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name: "name",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:     "cve",
							Cvss:    5,
							Summary: "Vuln summary",
						},
					},
				},
			},
		},
	}
}

func getPolicy() *storage.Policy {
	return &storage.Policy{
		Id:              "b3523d84-ac1a-4daa-a908-62d196c5a741",
		Name:            "Vulnerable Container",
		Categories:      []string{"Image Assurance", "Privileges Capabilities", "Container Configuration"},
		Description:     "Alert if the container contains vulnerabilities",
		Severity:        storage.Severity_LOW_SEVERITY,
		Rationale:       "This is the rationale",
		Remediation:     "This is the remediation",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		Scope: []*storage.Scope{
			{
				Cluster:   "prod cluster",
				Namespace: "stackrox",
				Label: &storage.Scope_Label{
					Key:   "com.docker.stack.namespace",
					Value: "prevent",
				},
			},
		},
		PolicyVersion: "1",
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Image Registry",
						Values: []*storage.PolicyValue{
							{
								Value: "docker.io",
							},
						},
					},
					{
						FieldName: "Image Remote",
						Values: []*storage.PolicyValue{
							{
								Value: "r/.*stackrox/nginx.*",
							},
						},
					},
					{
						FieldName: "Image Tag",
						Values: []*storage.PolicyValue{
							{
								Value: "1.10",
							},
						},
					},
					{
						FieldName: "Image Age",
						Values: []*storage.PolicyValue{
							{
								Value: "30",
							},
						},
					},
					{
						FieldName: "Dockerfile Line",
						Values: []*storage.PolicyValue{
							{
								Value: "VOLUME=/etc/*",
							},
						},
					},
					{
						FieldName: "CVE",
						Values: []*storage.PolicyValue{
							{
								Value: "CVE-1234",
							},
						},
					},
					{
						FieldName: "Image Component",
						Values: []*storage.PolicyValue{
							{
								Value: "berkeley*=.*",
							},
						},
					},
					{
						FieldName: "Image Scan Age",
						Values: []*storage.PolicyValue{
							{
								Value: "10",
							},
						},
					},
					{
						FieldName: "Environment Variable",
						Values: []*storage.PolicyValue{
							{
								Value: "UNSET=key=value",
							},
						},
					},
					{
						FieldName: "Volume Name",
						Values: []*storage.PolicyValue{
							{
								Value: "name",
							},
						},
					},
					{
						FieldName: "Volume Type",
						Values: []*storage.PolicyValue{
							{
								Value: "nfs",
							},
						},
					},
					{
						FieldName: "Volume Destination",
						Values: []*storage.PolicyValue{
							{
								Value: "/etc/network",
							},
						},
					},
					{
						FieldName: "Volume Source",
						Values: []*storage.PolicyValue{
							{
								Value: "10.0.0.1/export",
							},
						},
					},
					{
						FieldName: "Writable Mounted Volume",
						Values: []*storage.PolicyValue{
							{
								Value: "false",
							},
						},
					},
					{
						FieldName: "Port",
						Values: []*storage.PolicyValue{
							{
								Value: "8080",
							},
						},
					},
					{
						FieldName: "Protocol",
						Values: []*storage.PolicyValue{
							{
								Value: "tcp",
							},
						},
					},
					{
						FieldName: "Privileged",
						Values: []*storage.PolicyValue{
							{
								Value: "true",
							},
						},
					},
					{
						FieldName: "CVSS",
						Values: []*storage.PolicyValue{
							{
								Value: "\u003e= 5.000000",
							},
						},
					},
					{
						FieldName: "Drop Capabilities",
						Values: []*storage.PolicyValue{
							{
								Value: "DROP1",
							},
							{
								Value: "DROP2",
							},
						},
					},
					{
						FieldName: "Add Capabilities",
						Values: []*storage.PolicyValue{
							{
								Value: "ADD1",
							},
							{
								Value: "ADD2",
							},
						},
					},
				},
			},
		},
	}
}

func getAuditLogEventSourcePolicy() *storage.Policy {
	p := getPolicy()
	p.EventSource = storage.EventSource_AUDIT_LOG_EVENT
	// Limit scope to things that are supported by audit log event source
	p.Scope = []*storage.Scope{
		{
			Cluster:   "prod cluster",
			Namespace: "stackrox",
		},
	}
	// Only runtime policies can have audit log event source
	p.LifecycleStages = []storage.LifecycleStage{storage.LifecycleStage_RUNTIME}
	// Switch the policy values to things related to kube events
	p.PolicySections[0].PolicyGroups = []*storage.PolicyGroup{
		{
			FieldName: "Kubernetes Resource",
			Values:    []*storage.PolicyValue{{Value: "SECRETS"}},
		},
		{
			FieldName: "Kubernetes API Verb",
			Values:    []*storage.PolicyValue{{Value: "CREATE"}},
		},
	}
	return p
}
