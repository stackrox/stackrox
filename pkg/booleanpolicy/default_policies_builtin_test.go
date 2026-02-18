package booleanpolicy

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	policyUtils "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DefaultPoliciesTestSuite struct {
	basePoliciesTestSuite
}

func TestDefaultPolicies(t *testing.T) {
	t.Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	suite.Run(t, new(DefaultPoliciesTestSuite))
}

func (suite *DefaultPoliciesTestSuite) TestNoDuplicatePolicyIDs() {
	ids := set.NewStringSet()
	for _, p := range suite.defaultPolicies {
		suite.True(ids.Add(p.GetId()))
	}
}

func (suite *DefaultPoliciesTestSuite) TestRuntimePolicyFieldsCompile() {
	for _, p := range suite.defaultPolicies {
		if policyUtils.AppliesAtRunTime(p) {
			checkRegexCompiles(p.GetPolicySections(), fieldnames.ProcessName)
			checkRegexCompiles(p.GetPolicySections(), fieldnames.ProcessArguments)
			checkRegexCompiles(p.GetPolicySections(), fieldnames.ProcessAncestor)
		}
	}
}

func checkRegexCompiles(sections []*storage.PolicySection, fieldname string) {
	for _, s := range sections {
		for _, g := range s.GetPolicyGroups() {
			if g.GetFieldName() == fieldname {
				if policyVals := g.GetValues(); len(policyVals) > 0 {
					for _, policyVal := range policyVals {
						if v := policyVal.GetValue(); v != "" {
							regexp.MustCompile(v)
						}
					}
				}
			}
		}
	}
}

func (suite *DefaultPoliciesTestSuite) TestDefaultPolicies() {
	fixtureDep := fixtures.GetDeployment()
	fixturesImages := fixtures.DeploymentImages()

	suite.addDepAndImages(fixtureDep, fixturesImages...)

	nginx110 := &storage.Image{
		Id: "SHANGINX110",
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
			FullName: "docker.io/library/nginx:1.10",
		},
	}

	nginx110Dep := deploymentWithImage("nginx110", nginx110)
	suite.addDepAndImages(nginx110Dep, nginx110)

	oldScannedTime := time.Now().Add(-31 * 24 * time.Hour)

	oldScannedImage := &storage.Image{
		Id: "SHAOLDSCANNED",
		Name: &storage.ImageName{
			FullName: "docker.io/stackrox/old-scanned-image:0.1",
		},
		Scan: &storage.ImageScan{
			ScanTime: protoconv.ConvertTimeToTimestamp(oldScannedTime),
		},
	}
	oldScannedDep := deploymentWithImage("oldscanned", oldScannedImage)
	suite.addDepAndImages(oldScannedDep, oldScannedImage)

	addDockerFileImg := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "ADD",
			Value:       "deploy.sh",
		},
		{
			Instruction: "RUN",
			Value:       "deploy.sh",
		},
	})
	addDockerFileDep := deploymentWithImageAnyID(addDockerFileImg)
	suite.addDepAndImages(addDockerFileDep, addDockerFileImg)

	imagePort22Image := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "EXPOSE",
			Value:       "22/tcp",
		},
	})
	imagePort22Dep := deploymentWithImageAnyID(imagePort22Image)
	suite.addDepAndImages(imagePort22Dep, imagePort22Image)

	insecureCMDImage := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "CMD",
			Value:       "do an insecure thing",
		},
	})

	insecureCMDDep := deploymentWithImageAnyID(insecureCMDImage)
	suite.addDepAndImages(insecureCMDDep, insecureCMDImage)

	runSecretsImage := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "VOLUME",
			Value:       "/run/secrets",
		},
	})
	runSecretsArrayImage := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "VOLUME",
			Value:       "[/run/secrets]",
		},
	})
	runSecretsListImage := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "VOLUME",
			Value:       "/var/something /run/secrets",
		},
	})
	runSecretsArrayListImage := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "VOLUME",
			Value:       "[/var/something /run/secrets]",
		},
	})
	runSecretsDep := deploymentWithImageAnyID(runSecretsImage)
	runSecretsArrayDep := deploymentWithImageAnyID(runSecretsArrayImage)
	runSecretsListDep := deploymentWithImageAnyID(runSecretsListImage)
	runSecretsArrayListDep := deploymentWithImageAnyID(runSecretsArrayListImage)
	suite.addDepAndImages(runSecretsDep, runSecretsImage)
	suite.addDepAndImages(runSecretsArrayDep, runSecretsArrayImage)
	suite.addDepAndImages(runSecretsListDep, runSecretsListImage)
	suite.addDepAndImages(runSecretsArrayListDep, runSecretsArrayListImage)

	oldImageCreationTime := time.Now().Add(-100 * 24 * time.Hour)
	oldCreatedImage := &storage.Image{
		Id: "SHA:OLDCREATEDIMAGE",
		Name: &storage.ImageName{
			FullName: "docker.io/stackrox/old-image:0.1",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protoconv.ConvertTimeToTimestamp(oldImageCreationTime),
			},
		},
	}
	oldImageDep := deploymentWithImage("oldimagedep", oldCreatedImage)
	suite.addDepAndImages(oldImageDep, oldCreatedImage)

	apkImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "apk-tools", Version: "1.2"},
		{Name: "asfa", Version: "1.5"},
	})
	apkDep := deploymentWithImageAnyID(apkImage)
	suite.addDepAndImages(apkDep, apkImage)

	curlImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "curl", Version: "1.3"},
		{Name: "curlwithextra", Version: "0.9"},
	})
	curlDep := deploymentWithImageAnyID(curlImage)
	suite.addDepAndImages(curlDep, curlImage)

	componentDeps := make(map[string]*storage.Deployment)
	for _, component := range []string{"apt", "dnf", "wget"} {
		img := imageWithComponents([]*storage.EmbeddedImageScanComponent{
			{Name: component},
		})
		dep := deploymentWithImageAnyID(img)
		suite.addDepAndImages(dep, img)
		componentDeps[component] = dep
	}

	heartbleedDep := &storage.Deployment{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			{
				Name:            "nginx",
				SecurityContext: &storage.SecurityContext{Privileged: true},
				Image:           &storage.ContainerImage{Id: "HEARTBLEEDDEPSHA"},
			},
		},
	}
	suite.addDepAndImages(heartbleedDep, &storage.Image{
		Id:   "HEARTBLEEDDEPSHA",
		Name: &storage.ImageName{FullName: "heartbleed"},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.2"}},
				}},
			},
		},
	})

	requiredImageLabel := &storage.Deployment{
		Id: "requiredImageLabel",
		Containers: []*storage.Container{
			{
				Name:  "REQUIREDIMAGELABEL",
				Image: &storage.ContainerImage{Id: "requiredImageLabelImage"},
			},
		},
	}
	suite.addDepAndImages(requiredImageLabel, &storage.Image{
		Id: "requiredImageLabelImage",
		Name: &storage.ImageName{
			FullName: "docker.io/stackrox/required-image:0.1",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Labels: map[string]string{
					"required-label": "required-value",
				},
			},
		},
	})

	shellshockImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "shellshock", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2014-6271", Link: "https://shellshock", Cvss: 6},
			{Cve: "CVE-ARBITRARY", Link: "https://notshellshock"},
		}},
	})
	shellshockDep := deploymentWithImageAnyID(shellshockImage)
	suite.addDepAndImages(shellshockDep, shellshockImage)

	suppressedShellshockImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "shellshock", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2014-6271", Link: "https://shellshock", Cvss: 6, Suppressed: true},
			{Cve: "CVE-ARBITRARY", Link: "https://notshellshock"},
		}},
	})
	suppressedShellShockDep := deploymentWithImageAnyID(suppressedShellshockImage)
	suite.addDepAndImages(suppressedShellShockDep, suppressedShellshockImage)

	strutsImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "struts", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2017-5638", Link: "https://struts", Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.3"}},
		}},
		{Name: "OTHER", Version: "1.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-1223-451", Link: "https://cvefake"},
		}},
	})
	strutsDep := deploymentWithImageAnyID(strutsImage)
	suite.addDepAndImages(strutsDep, strutsImage)

	strutsImageSuppressed := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "struts", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2017-5638", Link: "https://struts", Suppressed: true, Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.3"}},
		}},
		{Name: "OTHER", Version: "1.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-1223-451", Link: "https://cvefake"},
		}},
	})
	strutsDepSuppressed := deploymentWithImageAnyID(strutsImageSuppressed)
	suite.addDepAndImages(strutsDepSuppressed, strutsImageSuppressed)

	// When image is pull out, the deferred field is set based upon the legacy suppressed field. Therefore, both are set.
	// However, here we are specifically testing whether detection is taking the new vulnerability state field into
	// account by not setting the suppressed field.
	structImageWithDeferredVulns := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "deferred-struts", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2017-5638", Link: "https://struts", State: storage.VulnerabilityState_DEFERRED, Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.3"}},
			{Cve: "CVE-2017-FP", Link: "https://struts", State: storage.VulnerabilityState_FALSE_POSITIVE, Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.3"}},
			{Cve: "CVE-2017-FAKE", Link: "https://struts", Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.3"}},
		}},
	})
	structDepWithDeferredVulns := deploymentWithImageAnyID(structImageWithDeferredVulns)
	suite.addDepAndImages(structDepWithDeferredVulns, structImageWithDeferredVulns)

	depWithNonSeriousVulnsImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "NOSERIOUS", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-1234-5678", Link: "https://abcdefgh"},
			{Cve: "CVE-5678-1234", Link: "https://lmnopqrst"},
		}},
	})
	depWithNonSeriousVulns := deploymentWithImageAnyID(depWithNonSeriousVulnsImage)
	suite.addDepAndImages(depWithNonSeriousVulns, depWithNonSeriousVulnsImage)

	dockerSockDep := &storage.Deployment{
		Id: "DOCKERSOCDEP",
		Containers: []*storage.Container{
			{
				Name: "dockersock",
				Volumes: []*storage.Volume{
					{Source: "/var/run/docker.sock", Name: "DOCKERSOCK", Type: "HostPath", ReadOnly: true},
					{Source: "NOTDOCKERSOCK"},
				}},
		},
	}
	suite.addDepAndImages(dockerSockDep)

	crioSockDep := &storage.Deployment{
		Id: "CRIOSOCDEP",
		Containers: []*storage.Container{
			{
				Name: "criosock",
				Volumes: []*storage.Volume{
					{Source: "/run/crio/crio.sock", Name: "CRIOSOCK", Type: "HostPath", ReadOnly: true},
					{Source: "NOTCRIORSOCK"},
				}},
		},
	}
	suite.addDepAndImages(crioSockDep)

	containerPort22Dep := &storage.Deployment{
		Id: "CONTAINERPORT22DEP",
		Ports: []*storage.PortConfig{
			{Protocol: "TCP", ContainerPort: 22},
			{Protocol: "UDP", ContainerPort: 4125},
		},
	}
	suite.addDepAndImages(containerPort22Dep)

	secretEnvDep := &storage.Deployment{
		Id: "SECRETENVDEP",
		Containers: []*storage.Container{
			{
				Name: "secretenv",
				Config: &storage.ContainerConfig{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{Key: "THIS_IS_SECRET_VAR", Value: "stealthmode", EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW},
						{Key: "HOME", Value: "/home/stackrox"},
					},
				}},
		},
	}
	suite.addDepAndImages(secretEnvDep)

	secretEnvSrcUnsetDep := &storage.Deployment{
		Id: "SECRETENVSRCUNSETDEP",
		Containers: []*storage.Container{
			{
				Name: "secretenvsrcunset",
				Config: &storage.ContainerConfig{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{Key: "THIS_IS_SECRET_VAR", Value: "stealthmode"},
					},
				}},
		},
	}
	suite.addDepAndImages(secretEnvSrcUnsetDep)

	secretKeyRefDep := &storage.Deployment{
		Id: "SECRETKEYREFDEP",
		Containers: []*storage.Container{
			{Config: &storage.ContainerConfig{
				Env: []*storage.ContainerConfig_EnvironmentConfig{
					{Key: "THIS_IS_SECRET_VAR", EnvVarSource: storage.ContainerConfig_EnvironmentConfig_SECRET_KEY},
					{Key: "HOME", Value: "/home/stackrox"},
				},
			}},
		},
	}
	suite.addDepAndImages(secretKeyRefDep)

	// Fake deployment that shouldn't match anything, just to make sure
	// that none of our queries will accidentally match it.
	suite.addDepAndImages(&storage.Deployment{Id: "FAKEID", Name: "FAKENAME"})

	depWithGoodEmailAnnotation := &storage.Deployment{
		Id: "GOODEMAILDEPID",
		Annotations: map[string]string{
			"email": "vv@stackrox.com",
		},
	}
	suite.addDepAndImages(depWithGoodEmailAnnotation)

	depWithOwnerAnnotation := &storage.Deployment{
		Id: "OWNERANNOTATIONDEP",
		Annotations: map[string]string{
			"owner": "IOWNTHIS",
			"blah":  "Blah",
		},
	}
	suite.addDepAndImages(depWithOwnerAnnotation)

	depWithOwnerLabel := &storage.Deployment{
		Id: "OWNERLABELDEP",
		Labels: map[string]string{
			"owner": "IOWNTHIS",
			"blah":  "Blah",
		},
	}
	suite.addDepAndImages(depWithOwnerLabel)

	depWitharbitraryAnnotations := &storage.Deployment{
		Id: "ARBITRARYANNOTATIONDEPID",
		Annotations: map[string]string{
			"emailnot": "vv@stackrox.com",
			"notemail": "vv@stackrox.com",
			"ownernot": "vv",
			"nowner":   "vv",
		},
	}
	suite.addDepAndImages(depWitharbitraryAnnotations)

	depWithBadEmailAnnotation := &storage.Deployment{
		Id: "BADEMAILDEPID",
		Annotations: map[string]string{
			"email": "NOTANEMAIL",
		},
	}
	suite.addDepAndImages(depWithBadEmailAnnotation)

	sysAdminDep := &storage.Deployment{
		Id: "SYSADMINDEPID",
		Containers: []*storage.Container{
			{
				Name: "cap-sys",
				SecurityContext: &storage.SecurityContext{
					AddCapabilities: []string{"SYS_ADMIN"},
				},
			},
		},
	}
	suite.addDepAndImages(sysAdminDep)

	depWithAllResourceLimitsRequestsSpecified := &storage.Deployment{
		Id: "ALLRESOURCESANDLIMITSDEP",
		Containers: []*storage.Container{
			{Resources: &storage.Resources{
				CpuCoresRequest: 0.1,
				CpuCoresLimit:   0.3,
				MemoryMbLimit:   100,
				MemoryMbRequest: 1251,
			}},
		},
	}
	suite.addDepAndImages(depWithAllResourceLimitsRequestsSpecified)

	depWithEnforcementBypassAnnotation := &storage.Deployment{
		Id: "ENFORCEMENTBYPASS",
		Annotations: map[string]string{
			"admission.stackrox.io/break-glass": "ticket-1234",
			"some-other":                        "annotation",
		},
	}
	suite.addDepAndImages(depWithEnforcementBypassAnnotation)

	hostMountDep := &storage.Deployment{
		Id: "HOSTMOUNT",
		Containers: []*storage.Container{
			{
				Name: "hostmount",
				Volumes: []*storage.Volume{
					{Source: "/etc/passwd", Name: "HOSTMOUNT", Type: "HostPath"},
					{Source: "/var/lib/kubelet", Name: "KUBELET", Type: "HostPath", ReadOnly: true},
				}},
		},
	}
	suite.addDepAndImages(hostMountDep)

	hostPIDDep := &storage.Deployment{
		Id:      "HOSTPID",
		HostPid: true,
	}
	suite.addDepAndImages(hostPIDDep)

	hostIPCDep := &storage.Deployment{
		Id:      "HOSTIPC",
		HostIpc: true,
	}
	suite.addDepAndImages(hostIPCDep)

	imgWithFixedByEmpty := suite.addImage(imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "EXplicitlyEmptyFixedBy", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-1234-5678", Cvss: 8, Link: "https://abcdefgh", SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{}},
		}},
	}))

	imgWithFixedByEmptyOnlyForSome := suite.addImage(imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "EXplicitlyEmptyFixedBy", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-1234-5678", Cvss: 8, Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, Link: "https://abcdefgh", SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{}},
		}},
		{Name: "Normal", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-5612-1245", Cvss: 8, Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, Link: "https://abcdefgh", SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "actually_fixable"}},
		}},
	}))

	rootUserImage := &storage.Image{
		Id: "SHA:ROOTUSERIMAGE",
		Name: &storage.ImageName{
			FullName: "docker.io/stackrox/rootuser:0.1",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				User: "root",
			},
		},
	}
	depWithRootUser := deploymentWithImageAnyID(rootUserImage)
	suite.addDepAndImages(depWithRootUser, rootUserImage)

	updateInstructionImage := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "RUN",
			Value:       "apt-get update",
		},
	})
	depWithUpdate := deploymentWithImageAnyID(updateInstructionImage)
	suite.addDepAndImages(depWithUpdate, updateInstructionImage)

	restrictedHostPortDep := &storage.Deployment{
		Id: "RESTRICTEDHOSTPORT",
		Ports: []*storage.PortConfig{
			{
				ExposureInfos: []*storage.PortConfig_ExposureInfo{
					{
						NodePort: 22,
					},
				},
			},
		},
	}

	suite.addDepAndImages(restrictedHostPortDep)

	mountPropagationDep := &storage.Deployment{
		Id: "MOUNTPROPAGATIONDEP",
		Containers: []*storage.Container{
			{
				Id: "MOUNTPROPAGATIONCONTAINER",
				Volumes: []*storage.Volume{
					{
						Name:             "ThisMountIsOnFire",
						MountPropagation: storage.Volume_BIDIRECTIONAL,
					},
				},
			},
		},
	}
	suite.addDepAndImages(mountPropagationDep)

	noSeccompProfileDep := &storage.Deployment{
		Id: "NOSECCOMPPROFILEDEP",
		Containers: []*storage.Container{
			{
				SecurityContext: &storage.SecurityContext{
					SeccompProfile: &storage.SecurityContext_SeccompProfile{
						Type: storage.SecurityContext_SeccompProfile_UNCONFINED,
					},
				},
			},
		},
	}
	suite.addDepAndImages(noSeccompProfileDep)

	hostNetworkDep := &storage.Deployment{
		Id:          "HOSTNETWORK",
		HostNetwork: true,
	}
	suite.addDepAndImages(hostNetworkDep)

	noAppArmorProfileDep := &storage.Deployment{
		Id: "NOAPPARMORPROFILEDEP",
		Containers: []*storage.Container{
			{
				Name: "No AppArmor Profile",
				Config: &storage.ContainerConfig{
					AppArmorProfile: "unconfined",
				},
			},
		},
	}
	suite.addDepAndImages(noAppArmorProfileDep)

	// Images "made by Red Hat" - coming from Red Hat registries or Red Hat remotes in quay.io
	registryAccessRedhatComUnverifiedImg := suite.imageWithSignatureVerificationResults("registry.access.redhat.com/redhat/ubi8:latest",
		[]*storage.ImageSignatureVerificationResult{
			{
				VerifierId: signatures.DefaultRedHatSignatureIntegration.GetId(),
				Status:     storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
			},
		},
	)
	registryRedHatIoUnverifiedImg := suite.imageWithSignatureVerificationResults("registry.redhat.io/redhat/ubi8:latest",
		[]*storage.ImageSignatureVerificationResult{
			{
				VerifierId: signatures.DefaultRedHatSignatureIntegration.GetId(),
				Status:     storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
			},
		},
	)

	quayOCPReleaseUnverifiedImg := suite.imageWithSignatureVerificationResults("quay.io/openshift-release-dev/ocp-release:latest",
		[]*storage.ImageSignatureVerificationResult{
			{
				VerifierId: signatures.DefaultRedHatSignatureIntegration.GetId(),
				Status:     storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
			},
		},
	)
	quayOCPArtDevUnverifiedImg := suite.imageWithSignatureVerificationResults("quay.io/openshift-release-dev/ocp-v4.0-art-dev:latest",
		[]*storage.ImageSignatureVerificationResult{
			{
				VerifierId: signatures.DefaultRedHatSignatureIntegration.GetId(),
				Status:     storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
			},
		},
	)

	suite.addImage(registryAccessRedhatComUnverifiedImg)
	suite.addImage(registryRedHatIoUnverifiedImg)
	suite.addImage(quayOCPReleaseUnverifiedImg)
	suite.addImage(quayOCPArtDevUnverifiedImg)

	// Index processes
	bashLineage := []string{"/bin/bash"}
	fixtureDepAptIndicator := suite.addIndicator(fixtureDep.GetId(), "apt", "", "/usr/bin/apt", bashLineage, 1)
	sysAdminDepAptIndicator := suite.addIndicator(sysAdminDep.GetId(), "apt", "install blah", "/usr/bin/apt", bashLineage, 1)

	kubeletIndicator := suite.addIndicator(containerPort22Dep.GetId(), "curl", "-v -k -SL https://12.13.14.15:10250", "/bin/curl", bashLineage, 1)
	kubeletIndicator2 := suite.addIndicator(containerPort22Dep.GetId(), "wget", "https://heapster.kube-system/metrics", "/bin/wget", bashLineage, 1)
	kubeletIndicator3 := suite.addIndicator(containerPort22Dep.GetId(), "curl", "https://12.13.14.15:10250 -v -k", "/bin/curl", bashLineage, 1)

	crontabIndicator := suite.addIndicator(containerPort22Dep.GetId(), "crontab", "1 2 3 4 5 6", "/bin/crontab", bashLineage, 1)

	nmapIndicatorfixtureDep1 := suite.addIndicator(fixtureDep.GetId(), "nmap", "blah", "/usr/bin/nmap", bashLineage, 1)
	nmapIndicatorfixtureDep2 := suite.addIndicator(fixtureDep.GetId(), "nmap", "blah2", "/usr/bin/nmap", bashLineage, 1)
	nmapIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "nmap", "", "/usr/bin/nmap", bashLineage, 1)

	ifconfigIndicatorfixtureDep1 := suite.addIndicator(fixtureDep.GetId(), "ifconfig", "blah", "/sbin/ifconfig", bashLineage, 1)
	ifconfigIndicatorfixtureDep2 := suite.addIndicator(fixtureDep.GetId(), "ifconfig", "blah2", "/usr/bin/ifconfig", bashLineage, 1)
	ipIndicatorfixtureDep := suite.addIndicator(fixtureDep.GetId(), "ip", "", "/sbin/ip", bashLineage, 1)
	arpIndicatorfixtureDep := suite.addIndicator(fixtureDep.GetId(), "arp", "", "/usr/sbin/arp", bashLineage, 1)
	ifconfigIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "ifconfig", "", "/sbin/ifconfig", bashLineage, 1)
	ipIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "ip", "", "/sbin/ip", bashLineage, 1)
	arpIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "arp", "", "/usr/sbin/arp", bashLineage, 1)
	// These two should not match for the Network Management Execution policy. See ROX-6011
	suite.addIndicator(fixtureDep.GetId(), "pip", "", "/usr/bin/pip", bashLineage, 1)
	suite.addIndicator(nginx110Dep.GetId(), "pip", "", "/usr/bin/pip", bashLineage, 1)

	javaLineage := []string{"/bin/bash", "/mnt/scripts/run_server.sh", "/bin/java"}
	fixtureDepJavaIndicator := suite.addIndicator(fixtureDep.GetId(), "/bin/bash", "-attack", "/bin/bash", javaLineage, 0)

	deploymentTestCases := []testCase{
		{
			policyName: "Latest tag",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message: "Container 'supervulnerable' has image with tag 'latest'",
					},
				},
			},
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				apkDep.GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'apk-tools' (version 1.2)",
					},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["apt"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'apt'",
					},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				curlDep.GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'curl' (version 1.3)",
					},
				},
			},
		},
		{
			policyName: "Red Hat Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["dnf"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'dnf'",
					},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["wget"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'wget'",
					},
				},
			},
		},
		{
			policyName: "Mount Container Runtime Socket",
			expectedViolations: map[string][]*storage.Alert_Violation{
				dockerSockDep.GetId(): {
					{
						Message: "Read-only volume 'DOCKERSOCK' has source '/var/run/docker.sock' and type 'HostPath'",
					},
				},
				crioSockDep.GetId(): {
					{
						Message: "Read-only volume 'CRIOSOCK' has source '/run/crio/crio.sock' and type 'HostPath'",
					},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				oldImageDep.GetId(): {
					{
						Message: fmt.Sprintf("Container 'oldimage' has image created at %s (UTC)", readable.Time(oldImageCreationTime)),
					},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				oldScannedDep.GetId(): {
					{
						Message: fmt.Sprintf("Container 'oldscannedimage' has image last scanned at %s (UTC)", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				imagePort22Dep.GetId(): {
					{
						Message: "Dockerfile line 'EXPOSE 22/tcp' found in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed",
			expectedViolations: map[string][]*storage.Alert_Violation{
				containerPort22Dep.GetId(): {
					{
						Message: "Exposed port 22/TCP is present",
					},
				},
			},
		},
		{
			policyName: "Privileged Container",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message: "Container 'nginx110container' is privileged",
					},
				},
				heartbleedDep.GetId(): {
					{
						Message: "Container 'nginx' is privileged",
					},
				},
			},
		},
		{
			policyName: "Container using read-write root filesystem",
			expectedViolations: map[string][]*storage.Alert_Violation{
				heartbleedDep.GetId(): {
					{
						Message: "Container 'nginx' uses a read-write root filesystem",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Container 'nginx110container' uses a read-write root filesystem",
					},
				},
				sysAdminDep.GetId(): {
					{
						Message: "Container 'cap-sys' uses a read-write root filesystem",
					},
				},
				noSeccompProfileDep.GetId(): {
					{
						Message: "Container  uses a read-write root filesystem",
					},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string][]*storage.Alert_Violation{
				insecureCMDDep.GetId(): {
					{
						Message: "Dockerfile line 'CMD do an insecure thing' found in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Improper Usage of Orchestrator Secrets Volume",
			expectedViolations: map[string][]*storage.Alert_Violation{
				runSecretsDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME /run/secrets' found in container 'ASFASF'",
					},
				},
				runSecretsArrayDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME [/run/secrets]' found in container 'ASFASF'",
					},
				},
				runSecretsListDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME /var/something /run/secrets' found in container 'ASFASF'",
					},
				},
				runSecretsArrayListDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME [/var/something /run/secrets]' found in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Images with no scans",
			shouldNotMatch: map[string]struct{}{
				// These deployments have scans on their images.
				fixtureDep.GetId():                 {},
				oldScannedDep.GetId():              {},
				heartbleedDep.GetId():              {},
				apkDep.GetId():                     {},
				curlDep.GetId():                    {},
				componentDeps["apt"].GetId():       {},
				componentDeps["dnf"].GetId():       {},
				componentDeps["wget"].GetId():      {},
				shellshockDep.GetId():              {},
				suppressedShellShockDep.GetId():    {},
				strutsDep.GetId():                  {},
				strutsDepSuppressed.GetId():        {},
				structDepWithDeferredVulns.GetId(): {},
				depWithNonSeriousVulns.GetId():     {},
				// The rest of the deployments have no images!
				"FAKEID":                                          {},
				containerPort22Dep.GetId():                        {},
				dockerSockDep.GetId():                             {},
				crioSockDep.GetId():                               {},
				secretEnvDep.GetId():                              {},
				secretEnvSrcUnsetDep.GetId():                      {},
				secretKeyRefDep.GetId():                           {},
				depWithOwnerAnnotation.GetId():                    {},
				depWithOwnerLabel.GetId():                         {},
				depWithGoodEmailAnnotation.GetId():                {},
				depWithBadEmailAnnotation.GetId():                 {},
				depWitharbitraryAnnotations.GetId():               {},
				sysAdminDep.GetId():                               {},
				depWithAllResourceLimitsRequestsSpecified.GetId(): {},
				depWithEnforcementBypassAnnotation.GetId():        {},
				hostMountDep.GetId():                              {},
				restrictedHostPortDep.GetId():                     {},
				hostPIDDep.GetId():                                {},
				hostIPCDep.GetId():                                {},
				mountPropagationDep.GetId():                       {},
				noSeccompProfileDep.GetId():                       {},
				hostNetworkDep.GetId():                            {},
				noAppArmorProfileDep.GetId():                      {},
			},
			sampleViolationForMatched: "Image in container '%s' has not been scanned",
		},
		{
			policyName: "Required Annotation: Email",
			shouldNotMatch: map[string]struct{}{
				depWithGoodEmailAnnotation.GetId(): {},
			},
			sampleViolationForMatched: "Required annotation not found (key = 'email', value = '[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+')",
		},
		{
			policyName: "Required Label: Owner/Team",
			shouldNotMatch: map[string]struct{}{
				depWithOwnerLabel.GetId(): {},
				fixtureDep.GetId():        {},
			},
			sampleViolationForMatched: "Required label not found (key = 'owner|team', value = '.+')",
		},
		{
			policyName: "Required Annotation: Owner/Team",
			shouldNotMatch: map[string]struct{}{
				depWithOwnerAnnotation.GetId(): {},
				fixtureDep.GetId():             {},
			},
			sampleViolationForMatched: "Required annotation not found (key = 'owner|team', value = '.+')",
		},
		{
			policyName: "CAP_SYS_ADMIN capability added",
			expectedViolations: map[string][]*storage.Alert_Violation{
				sysAdminDep.GetId(): {
					{
						Message: "Container 'cap-sys' adds capability SYS_ADMIN",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Container 'nginx110container' adds capability SYS_ADMIN",
					},
				},
			},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2) in container 'ASFASF'",
					},
				},
				// CVE-2017-5638 is deferred in `deferred-struct`, hence no violation.
			},
		},
		{
			policyName: "No CPU request or memory limit specified",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetId(): {
					{Message: "Memory limit set to 0 MB for container 'nginx110container'"},
				},
			},
		},
		{
			policyName: "Environment Variable Contains Secret",
			expectedViolations: map[string][]*storage.Alert_Violation{
				secretEnvDep.GetId(): {
					{
						Message: "Environment variable 'THIS_IS_SECRET_VAR' is present in container 'secretenv'",
					},
				},
			},
		},
		{
			policyName: "Secret Mounted as Environment Variable",
			expectedViolations: map[string][]*storage.Alert_Violation{
				secretKeyRefDep.GetId(): {
					{
						Message: "Environment variable 'THIS_IS_SECRET_VAR' is present and references a Secret",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 6 and Privileged",
			expectedViolations: map[string][]*storage.Alert_Violation{
				heartbleedDep.GetId(): {
					{
						Message: "Container 'nginx' is privileged",
					},
					{
						Message: "Fixable CVE-2014-0160 (CVSS 6) (severity Unknown) found in component 'heartbleed' (version 1.2) in container 'nginx', resolved by version v1.2",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 7",
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
				structDepWithDeferredVulns.GetId(): {
					{
						Message: "Fixable CVE-2017-FAKE (CVSS 8) (severity Important) found in component 'deferred-struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
			},
		},
		{
			policyName: "Fixable Severity at least Important",
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
				structDepWithDeferredVulns.GetId(): {
					{
						Message: "Fixable CVE-2017-FAKE (CVSS 8) (severity Important) found in component 'deferred-struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string][]*storage.Alert_Violation{
				addDockerFileDep.GetId(): {
					{
						Message: "Dockerfile line 'ADD deploy.sh' found in container 'ASFASF'",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Dockerfile line 'ADD FILE:blah' found in container 'nginx110container'",
					},
					{
						Message: "Dockerfile line 'ADD file:4eedf861fb567fffb2694b65ebd...' found in container 'supervulnerable'",
					},
				},
			},
		},
		{
			policyName: "nmap Execution",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId():  {nmapIndicatorfixtureDep1, nmapIndicatorfixtureDep2},
				nginx110Dep.GetId(): {nmapIndicatorNginx110Dep},
			},
		},
		{
			policyName: "Process Targeting Cluster Kubelet Endpoint",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				containerPort22Dep.GetId(): {kubeletIndicator, kubeletIndicator2, kubeletIndicator3},
			},
		},
		{
			policyName: "crontab Execution",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				containerPort22Dep.GetId(): {crontabIndicator},
			},
		},
		{
			policyName: "Ubuntu Package Manager Execution",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId():  {fixtureDepAptIndicator},
				sysAdminDep.GetId(): {sysAdminDepAptIndicator},
			},
		},
		{
			policyName: "Process with UID 0",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId(): {fixtureDepJavaIndicator},
			},
		},
		{
			policyName: "Shell Spawned by Java Application",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId(): {fixtureDepJavaIndicator},
			},
		},
		{
			policyName: "Network Management Execution",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId():  {ifconfigIndicatorfixtureDep1, ifconfigIndicatorfixtureDep2, ipIndicatorfixtureDep, arpIndicatorfixtureDep},
				nginx110Dep.GetId(): {ifconfigIndicatorNginx110Dep, ipIndicatorNginx110Dep, arpIndicatorNginx110Dep},
			},
		},
		{
			policyName: "Emergency Deployment Annotation",
			expectedViolations: map[string][]*storage.Alert_Violation{
				depWithEnforcementBypassAnnotation.GetId(): {
					{Message: "Disallowed annotations found: admission.stackrox.io/break-glass=ticket-1234"},
				},
			},
		},
		{
			policyName: "Mounting Sensitive Host Directories",
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostMountDep.GetId(): {
					{Message: "Read-only volume 'KUBELET' has source '/var/lib/kubelet' and type 'HostPath'"},
					{Message: "Writable volume 'HOSTMOUNT' has source '/etc/passwd' and type 'HostPath'"},
				},
				dockerSockDep.GetId(): {
					{Message: "Read-only volume 'DOCKERSOCK' has source '/var/run/docker.sock' and type 'HostPath'"},
				},
			},
		},
		{
			policyName: writableHostMountPolicyName,
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostMountDep.GetId(): {
					{Message: "Writable volume 'HOSTMOUNT' has source '/etc/passwd' and type 'HostPath'"},
				},
			},
		},
		{
			policyName: "Docker CIS 4.1: Ensure That a User for the Container Has Been Created",
			expectedViolations: map[string][]*storage.Alert_Violation{
				depWithRootUser.GetId(): {
					{
						Message: "Container 'rootuser' has image with user 'root'",
					},
				},
			},
		},
		{
			policyName: "Docker CIS 4.7: Alert on Update Instruction",
			expectedViolations: map[string][]*storage.Alert_Violation{
				depWithUpdate.GetId(): {
					{
						Message: "Dockerfile line 'RUN apt-get update' found in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Docker CIS 5.7: Ensure privileged ports are not mapped within containers",
			expectedViolations: map[string][]*storage.Alert_Violation{
				restrictedHostPortDep.GetId(): {
					{
						Message: "Exposed node port 22 is present",
					},
				},
			},
		},
		{
			policyName: "Docker CIS 5.15: Ensure that the host's process namespace is not shared",
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostPIDDep.GetId(): {
					{Message: "Deployment uses the host's process ID namespace"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.16: Ensure that the host's IPC namespace is not shared",
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostIPCDep.GetId(): {
					{Message: "Deployment uses the host's IPC namespace"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.19: Ensure mount propagation mode is not enabled",
			expectedViolations: map[string][]*storage.Alert_Violation{
				mountPropagationDep.GetId(): {
					{Message: "Writable volume 'ThisMountIsOnFire' has mount propagation 'bidirectional'"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.21: Ensure the default seccomp profile is not disabled",
			expectedViolations: map[string][]*storage.Alert_Violation{
				noSeccompProfileDep.GetId(): {
					{Message: "Container has Seccomp profile type 'unconfined'"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.9 and 5.20: Ensure that the host's network namespace is not shared",
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostNetworkDep.GetId(): {
					{Message: "Deployment uses the host's network namespace"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.1 Ensure that, if applicable, an AppArmor Profile is enabled",
			expectedViolations: map[string][]*storage.Alert_Violation{
				noAppArmorProfileDep.GetId(): {
					{Message: "Container 'No AppArmor Profile' has AppArmor profile type 'unconfined'"},
				},
			},
		},
		{
			policyName:                 "Docker CIS 4.4: Ensure images are scanned and rebuilt to include security patches",
			allowUnvalidatedViolations: true,
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
				heartbleedDep.GetId(): {
					{
						Message: "Fixable CVE-2014-0160 (CVSS 6) (severity Unknown) found in component 'heartbleed' (version 1.2) in container 'nginx', resolved by version v1.2",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Fixable CVE-2014-6200 (CVSS 5) (severity Moderate) found in component 'name' (version 1.2.3.4) in container 'supervulnerable', resolved by version abcdefg",
					},
				},
				fixtures.LightweightDeployment().GetId(): {
					{
						Message: "Fixable CVE-2014-6200 (CVSS 5) (severity Moderate) found in component 'name' (version 1.2.3.4) in container 'supervulnerable', resolved by version abcdefg",
					},
				},
				structDepWithDeferredVulns.GetId(): {
					{
						Message: "Fixable CVE-2017-FAKE (CVSS 8) (severity Important) found in component 'deferred-struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
			},
		},
		{
			policyName: anyHostPathPolicyName,
			expectedViolations: map[string][]*storage.Alert_Violation{
				dockerSockDep.GetId(): {
					{Message: "Read-only volume 'DOCKERSOCK' has source '/var/run/docker.sock' and type 'HostPath'"},
				},
				crioSockDep.GetId(): {
					{Message: "Read-only volume 'CRIOSOCK' has source '/run/crio/crio.sock' and type 'HostPath'"},
				},
				hostMountDep.GetId(): {
					{Message: "Read-only volume 'KUBELET' has source '/var/lib/kubelet' and type 'HostPath'"},
					{Message: "Writable volume 'HOSTMOUNT' has source '/etc/passwd' and type 'HostPath'"},
				},
			},
		},
	}

	for _, c := range deploymentTestCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(fmt.Sprintf("%s (on deployments)", c.policyName), func(t *testing.T) {
			if len(c.shouldNotMatch) == 0 {
				assert.True(t, (c.expectedViolations != nil) != (c.expectedProcessViolations != nil), "Every test case must "+
					"contain exactly one of expectedViolations and expectedProcessViolations")
			} else {
				assert.Nil(t, c.expectedViolations, "Cannot specify shouldNotMatch AND expectedViolations")
				assert.Nil(t, c.expectedProcessViolations, "Cannot specify shouldNotMatch AND expectedProcessViolations")
			}

			m, err := BuildDeploymentMatcher(p)
			require.NoError(t, err)

			if c.expectedProcessViolations != nil {
				processMatcher, err := BuildDeploymentWithProcessMatcher(p)
				require.NoError(t, err)
				for deploymentID, processes := range c.expectedProcessViolations {
					expectedProcesses := set.NewStringSet(sliceutils.Map(processes, func(p *storage.ProcessIndicator) string {
						return p.GetId()
					})...)
					deployment := suite.deployments[deploymentID]

					for _, process := range suite.deploymentsToIndicators[deploymentID] {
						match := getViolationsWithAndWithoutCaching(t, func(cache *CacheReceptacle) (Violations, error) {
							return processMatcher.MatchDeploymentWithProcess(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)), process, false)
						})
						require.NoError(t, err)
						if expectedProcesses.Contains(process.GetId()) {
							assert.NotNil(t, match.ProcessViolation, "process %+v should match", process)
						} else {
							assert.Nil(t, match.ProcessViolation, "process %+v should not match", process)
						}
					}
				}
				return
			}

			actualViolations := make(map[string][]*storage.Alert_Violation)
			for id, deployment := range suite.deployments {
				violationsForDep := getViolationsWithAndWithoutCaching(t, func(cache *CacheReceptacle) (Violations, error) {
					return m.MatchDeployment(cache, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
				})
				assert.Nil(t, violationsForDep.ProcessViolation)
				if alertViolations := violationsForDep.AlertViolations; len(alertViolations) > 0 {
					actualViolations[id] = alertViolations
				}
			}
			if len(c.shouldNotMatch) > 0 {
				for shouldNotMatchID := range c.shouldNotMatch {
					assert.Contains(t, suite.deployments, shouldNotMatchID)
					assert.NotContains(t, actualViolations, shouldNotMatchID)
				}
				for id := range suite.deployments {
					if _, shouldNotMatch := c.shouldNotMatch[id]; !shouldNotMatch {
						assert.Contains(t, actualViolations, id)

						// TODO(rc) update for BPL and check all sampleViolationForMatched
						if c.policyName == "Images with no scans" {
							if len(suite.deployments[id].GetContainers()) == 1 {
								msg := fmt.Sprintf(c.sampleViolationForMatched, suite.deployments[id].GetContainers()[0].GetName())
								protoassert.SlicesEqual(t, actualViolations[id], []*storage.Alert_Violation{{Message: msg}})
							}
						}
					}
				}
				return
			}

			for id := range suite.deployments {
				violations, expected := c.expectedViolations[id]
				if expected {
					assert.Contains(t, actualViolations, id)

					if c.allowUnvalidatedViolations {
						assert.NotEmpty(t, violations)
						for _, violation := range violations {
							protoassert.SliceContains(t, actualViolations[id], violation)
						}
					} else {
						protoassert.SlicesEqual(t, violations, actualViolations[id])
					}
				} else {
					assert.NotContains(t, actualViolations, id)
				}
			}

		})
	}

	imageTestCases := []testCase{
		{
			policyName: "Latest tag",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{Message: "Image has tag 'latest'"},
				},
			},
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(apkDep): {
					{
						Message: "Image includes component 'apk-tools' (version 1.2)",
					},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["apt"]): {
					{
						Message: "Image includes component 'apt'",
					},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(curlDep): {
					{
						Message: "Image includes component 'curl' (version 1.3)",
					},
				},
			},
		},
		{
			policyName: "Red Hat Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["dnf"]): {
					{
						Message: "Image includes component 'dnf'",
					},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["wget"]): {
					{
						Message: "Image includes component 'wget'",
					},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(oldImageDep): {
					{
						Message: fmt.Sprintf("Image was created at %s (UTC)", readable.Time(oldImageCreationTime)),
					},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(oldScannedDep): {
					{
						Message: fmt.Sprintf("Image was last scanned at %s (UTC)", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(imagePort22Dep): {
					{
						Message: "Dockerfile line 'EXPOSE 22/tcp' found",
					},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(insecureCMDDep): {
					{
						Message: "Dockerfile line 'CMD do an insecure thing' found",
					},
				},
			},
		},
		{
			policyName: "Improper Usage of Orchestrator Secrets Volume",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(runSecretsDep): {
					{
						Message: "Dockerfile line 'VOLUME /run/secrets' found",
					},
				},
				suite.imageIDFromDep(runSecretsArrayDep): {
					{
						Message: "Dockerfile line 'VOLUME [/run/secrets]' found",
					},
				},
				suite.imageIDFromDep(runSecretsListDep): {
					{
						Message: "Dockerfile line 'VOLUME /var/something /run/secrets' found",
					},
				},
				suite.imageIDFromDep(runSecretsArrayListDep): {
					{
						Message: "Dockerfile line 'VOLUME [/var/something /run/secrets]' found",
					},
				},
			},
		},
		{
			policyName: "Images with no scans",
			shouldNotMatch: map[string]struct{}{
				oldScannedImage.GetId():                          {},
				suite.imageIDFromDep(heartbleedDep):              {},
				apkImage.GetId():                                 {},
				curlImage.GetId():                                {},
				suite.imageIDFromDep(componentDeps["apt"]):       {},
				suite.imageIDFromDep(componentDeps["dnf"]):       {},
				suite.imageIDFromDep(componentDeps["wget"]):      {},
				shellshockImage.GetId():                          {},
				suppressedShellshockImage.GetId():                {},
				strutsImage.GetId():                              {},
				strutsImageSuppressed.GetId():                    {},
				structImageWithDeferredVulns.GetId():             {},
				depWithNonSeriousVulnsImage.GetId():              {},
				fixtureDep.GetContainers()[0].GetImage().GetId(): {},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {},
				suite.imageIDFromDep(oldScannedDep):              {},
				imgWithFixedByEmpty.GetId():                      {},
				imgWithFixedByEmptyOnlyForSome.GetId():           {},
			},
			sampleViolationForMatched: "Image has not been scanned",
			expectedViolations:        map[string][]*storage.Alert_Violation{},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2)",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 7",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2), resolved by version v1.3",
					},
				},
				imgWithFixedByEmptyOnlyForSome.GetId(): {
					{
						Message: "Fixable CVE-5612-1245 (CVSS 8) (severity Critical) found in component 'Normal' (version 2.3), resolved by version actually_fixable",
					},
				},
			},
		},
		{
			policyName: "Fixable Severity at least Important",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2), resolved by version v1.3",
					},
				},
				imgWithFixedByEmptyOnlyForSome.GetId(): {
					{
						Message: "Fixable CVE-5612-1245 (CVSS 8) (severity Critical) found in component 'Normal' (version 2.3), resolved by version actually_fixable",
					},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(addDockerFileDep): {
					{
						Message: "Dockerfile line 'ADD deploy.sh' found",
					},
				},
				fixtureDep.GetContainers()[0].GetImage().GetId(): {
					{
						Message: "Dockerfile line 'ADD FILE:blah' found",
					},
				},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{
						Message: "Dockerfile line 'ADD file:4eedf861fb567fffb2694b65ebd...' found",
					},
				},
			},
		},
		{
			policyName: "Required Image Label",
			shouldNotMatch: map[string]struct{}{
				"requiredImageLabelImage": {},
			},
			sampleViolationForMatched: "Required label not found (found labels: <empty>)",
		},

		{
			// We can only test that the policy triggers for unverified images. The "shouldNotMatch" field cannot be
			// used to verify that signed images don't trigger violations, because then the logic expects that all
			// other images (not listed in shouldNotMatch) trigger a violation; and in this case only unsigned images
			// in Red Hat registries trigger violations - any other unsigned images are fine and should not trigger.
			policyName: "Red Hat images must be signed by a Red Hat release key",
			expectedViolations: map[string][]*storage.Alert_Violation{
				registryRedHatIoUnverifiedImg.GetId(): {
					{
						Message: "Image has registry 'registry.redhat.io'",
					},
					{
						Message: "Image signature is not verified by the specified signature integration(s).",
					},
				},
				registryAccessRedhatComUnverifiedImg.GetId(): {
					{
						Message: "Image has registry 'registry.access.redhat.com'",
					},
					{
						Message: "Image signature is not verified by the specified signature integration(s).",
					},
				},
				quayOCPReleaseUnverifiedImg.GetId(): {
					{
						Message: "Image has registry 'quay.io' and remote 'openshift-release-dev/ocp-release'",
					},
					{
						Message: "Image signature is not verified by the specified signature integration(s).",
					},
				},
				quayOCPArtDevUnverifiedImg.GetId(): {
					{
						Message: "Image has registry 'quay.io' and remote 'openshift-release-dev/ocp-v4.0-art-dev'",
					},
					{
						Message: "Image signature is not verified by the specified signature integration(s).",
					},
				},
			},
		},
	}

	for _, c := range imageTestCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(fmt.Sprintf("%s (on images)", c.policyName), func(t *testing.T) {
			assert.Nil(t, c.expectedProcessViolations)

			m, err := BuildImageMatcher(p)
			require.NoError(t, err)

			actualViolations := make(map[string][]*storage.Alert_Violation)
			for id, image := range suite.images {
				violationsForImg := getViolationsWithAndWithoutCaching(t, func(cache *CacheReceptacle) (Violations, error) {
					return m.MatchImage(cache, image)
				})
				suite.Nil(violationsForImg.ProcessViolation)
				if alertViolations := violationsForImg.AlertViolations; len(alertViolations) > 0 {
					actualViolations[id] = alertViolations
				}
			}

			for id, violations := range c.expectedViolations {
				assert.Contains(t, actualViolations, id)
				protoassert.SlicesEqual(t, violations, actualViolations[id])
			}
			if len(c.shouldNotMatch) > 0 {
				if c.policyName == "Required Image Label" {
					for id, image := range suite.images {
						if image.GetMetadata() == nil {
							c.shouldNotMatch[id] = struct{}{}
						}
					}
				}

				for shouldNotMatchID := range c.shouldNotMatch {
					assert.Contains(t, suite.images, shouldNotMatchID, "%s is not a known image id in the suite", shouldNotMatchID)
					assert.NotContains(t, actualViolations, shouldNotMatchID)
				}

				for id := range suite.images {
					if _, shouldNotMatch := c.shouldNotMatch[id]; !shouldNotMatch {
						assert.Contains(t, actualViolations, id)
						protoassert.SlicesEqual(t, actualViolations[id], []*storage.Alert_Violation{{Message: c.sampleViolationForMatched}})
					}
				}
			}
		})
	}
}
