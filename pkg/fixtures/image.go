package fixtures

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

func getVulnsPerComponent(componentIndex int) []*storage.Vulnerability {
	numVulnsPerComponent := 5
	vulnsPerComponent := make([]*storage.Vulnerability, 0, numVulnsPerComponent)
	for i := 0; i < numVulnsPerComponent; i++ {
		cveName := fmt.Sprintf("CVE-2014-62%d%d", componentIndex, i)
		vulnsPerComponent = append(vulnsPerComponent, &storage.Vulnerability{
			Cve:     cveName,
			Cvss:    5,
			Summary: "GNU Bash through 4.3 processes trailing strings after function definitions in the values of environment variables, which allows remote attackers to execute arbitrary code via a crafted environment, as demonstrated by vectors involving the ForceCommand feature in OpenSSH sshd, the mod_cgi and mod_cgid modules in the Apache HTTP Server, scripts executed by unspecified DHCP clients, and other situations in which setting the environment occurs across a privilege boundary from Bash execution, aka \"ShellShock.\"  NOTE: the original fix for this issue was incorrect; CVE-2014-7169 has been assigned to cover the vulnerability that is still present after the incorrect fix.",
			Link:    fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cveName),
			SetFixedBy: &storage.Vulnerability_FixedBy{
				FixedBy: "abcdefg",
			},
		})
	}
	return vulnsPerComponent
}

// GetImage returns a Mock Image
func GetImage() *storage.Image {
	numComponentsPerImage := 50
	componentsPerImage := make([]*storage.ImageScanComponent, 0, numComponentsPerImage)
	for i := 0; i < numComponentsPerImage; i++ {
		componentsPerImage = append(componentsPerImage, &storage.ImageScanComponent{
			Name:    "name",
			Version: "1.2.3.4",
			License: &storage.License{
				Name: "blah",
				Type: "GPL",
			},
			Vulns: getVulnsPerComponent(i),
		})
	}
	author := "author"
	return &storage.Image{
		Id: "sha256:SHA2",
		Name: &storage.ImageName{
			Registry: "stackrox.io",
			Remote:   "srox/mongo",
			Tag:      "latest",
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
