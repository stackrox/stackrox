// +build integration

package docker

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// DefaultRegistry defaults to dockerhub
	defaultRegistry = "https://registry-1.docker.io" // variable so that it could be potentially changed
)

func TestGetMetadataIntegration(t *testing.T) {
	url := defaultRegistry
	username := ""
	password := ""

	dockerHubClient := &dockerRegistry{
		url:      url,
		username: username,
		password: password,
	}

	image := v1.Image{
		Name: &v1.ImageName{
			Remote: "library/nginx",
			Tag:    "1.10",
		},
	}
	metadata, err := dockerHubClient.Metadata(&image)
	require.Nil(t, err)

	author := `NGINX Docker Maintainers "docker-maint@nginx.com"`

	expectedMetadata := &v1.ImageMetadata{
		Author:  author,
		Created: getProtoTimestamp(1490644114817993292),
		Layers: []*v1.ImageLayer{
			{
				Instruction: "CMD",
				Value:       `["nginx" "-g" "daemon off;"]`,
				Author:      author,
				Created:     getProtoTimestamp(1490644114817993292),
			},
			{
				Instruction: "EXPOSE",
				Value:       "443/tcp 80/tcp",
				Author:      author,
				Created:     getProtoTimestamp(1490644114499698303),
			},
			{
				Instruction: "RUN",
				Value:       "ln -sf /dev/stdout /var/log/nginx/access.log && ln -sf /dev/stderr /var/log/nginx/error.log",
				Author:      author,
				Created:     getProtoTimestamp(1490644114191218214),
			},
			{
				Instruction: "RUN",
				Value:       `apt-key adv --keyserver hkp://pgp.mit.edu:80 --recv-keys 573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62 && echo "deb http://nginx.org/packages/debian/ jessie nginx" >> /etc/apt/sources.list && apt-get update && apt-get install --no-install-recommends --no-install-suggests -y ca-certificates nginx=${NGINX_VERSION} nginx-module-xslt nginx-module-geoip nginx-module-image-filter nginx-module-perl nginx-module-njs gettext-base && rm -rf /var/lib/apt/lists/*`,
				Author:      author,
				Created:     getProtoTimestamp(1490644113325920681),
			},
			{
				Instruction: "ENV",
				Value:       "NGINX_VERSION=1.10.3-1~jessie",
				Author:      author,
				Created:     getProtoTimestamp(1490644099151777495),
			},
			{
				Instruction: "MAINTAINER",
				Value:       author,
				Author:      author,
				Created:     getProtoTimestamp(1490134018207888376),
			},
			{
				Instruction: "CMD",
				Value:       `["/bin/bash"]`,
				Created:     getProtoTimestamp(1490120945091744235),
			},
			{
				Instruction: "ADD",
				Value:       "file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /",
				Created:     getProtoTimestamp(1490120931055495122),
			},
		},
	}
	assert.Equal(t, expectedMetadata, metadata)
}
