package docker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	manifestV1 "github.com/docker/distribution/manifest/schema1"
	ptypes "github.com/gogo/protobuf/types"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var server *httptest.Server

const getMetadataPayload = `{
   "schemaVersion": 1,
   "name": "library/nginx",
   "tag": "1.10",
   "architecture": "amd64",
   "fsLayers": [
      {
         "blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
      },
      {
         "blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
      },
      {
         "blobSum": "sha256:556c62bb43ac9073f4dfc95383e83f8048633a041cb9e7eb2c1f346ba39a5183"
      },
      {
         "blobSum": "sha256:1e3e18a64ea9924fd9688d125c2844c4df144e41b1d2880a06423bca925b778c"
      },
      {
         "blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
      },
      {
         "blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
      },
      {
         "blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
      },
      {
         "blobSum": "sha256:6d827a3ef358f4fa21ef8251f95492e667da826653fd43641cef5a877dc03a70"
      }
   ],
   "history": [
      {
         "v1Compatibility": "{\"architecture\":\"amd64\",\"author\":\"NGINX Docker Maintainers \\\"docker-maint@nginx.com\\\"\",\"Config\":{\"Hostname\":\"7e9ec6cde4d1\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"ExposedPorts\":{\"443/tcp\":{},\"80/tcp\":{}},\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\",\"NGINX_VERSION=1.10.3-1~jessie\"],\"Cmd\":[\"nginx\",\"-g\",\"daemon off;\"],\"ArgsEscaped\":true,\"Image\":\"sha256:487d6abc81226d443492de061719d10329dfb4107a7621eed15558779e960b6b\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":null,\"OnBuild\":[],\"Labels\":{}},\"container\":\"e012082cd2cf5748d7a181bd5f207822fc6d2628f8ec8b1f00619df5f8ac1c4c\",\"container_config\":{\"Hostname\":\"7e9ec6cde4d1\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"ExposedPorts\":{\"443/tcp\":{},\"80/tcp\":{}},\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\",\"NGINX_VERSION=1.10.3-1~jessie\"],\"Cmd\":[\"/bin/sh\",\"-c\",\"#(nop) \",\"CMD [\\\"nginx\\\" \\\"-g\\\" \\\"daemon off;\\\"]\"],\"ArgsEscaped\":true,\"Image\":\"sha256:487d6abc81226d443492de061719d10329dfb4107a7621eed15558779e960b6b\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":null,\"OnBuild\":[],\"Labels\":{}},\"created\":\"2017-03-27T19:48:34.817993292Z\",\"docker_version\":\"1.12.6\",\"id\":\"5cd84ba04b60f6634ae61595be4828944c0fdb65fd1e7b45e6ba7262a9c034ed\",\"os\":\"linux\",\"parent\":\"a75af3100369df49da6dbe8e03cd364fa27e99f8871e7c959e48898438bebc17\",\"throwaway\":true}"
      },
      {
         "v1Compatibility": "{\"id\":\"a75af3100369df49da6dbe8e03cd364fa27e99f8871e7c959e48898438bebc17\",\"parent\":\"42fb678ea343a9935e8eead3f7dda21f6fada31830a6c25345af7fdcc47447fd\",\"created\":\"2017-03-27T19:48:34.499698303Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop)  EXPOSE 443/tcp 80/tcp\"]},\"author\":\"NGINX Docker Maintainers \\\"docker-maint@nginx.com\\\"\",\"throwaway\":true}"
      },
      {
         "v1Compatibility": "{\"id\":\"42fb678ea343a9935e8eead3f7dda21f6fada31830a6c25345af7fdcc47447fd\",\"parent\":\"409da06c5af5c476ae322c4dee60affd85933f06b4ca79f2b08dc81412a37e14\",\"created\":\"2017-03-27T19:48:34.191218214Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c ln -sf /dev/stdout /var/log/nginx/access.log \\t\\u0026\\u0026 ln -sf /dev/stderr /var/log/nginx/error.log\"]},\"author\":\"NGINX Docker Maintainers \\\"docker-maint@nginx.com\\\"\"}"
      },
      {
         "v1Compatibility": "{\"id\":\"409da06c5af5c476ae322c4dee60affd85933f06b4ca79f2b08dc81412a37e14\",\"parent\":\"5cf74bcb1bde2e2249824a682f45235954543a5d57081db22c96402342db49e9\",\"created\":\"2017-03-27T19:48:33.325920681Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c apt-key adv --keyserver hkp://pgp.mit.edu:80 --recv-keys 573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62 \\t\\u0026\\u0026 echo \\\"deb http://nginx.org/packages/debian/ jessie nginx\\\" \\u003e\\u003e /etc/apt/sources.list \\t\\u0026\\u0026 apt-get update \\t\\u0026\\u0026 apt-get install --no-install-recommends --no-install-suggests -y \\t\\t\\t\\t\\t\\tca-certificates \\t\\t\\t\\t\\t\\tnginx=${NGINX_VERSION} \\t\\t\\t\\t\\t\\tnginx-module-xslt \\t\\t\\t\\t\\t\\tnginx-module-geoip \\t\\t\\t\\t\\t\\tnginx-module-image-filter \\t\\t\\t\\t\\t\\tnginx-module-perl \\t\\t\\t\\t\\t\\tnginx-module-njs \\t\\t\\t\\t\\t\\tgettext-base \\t\\u0026\\u0026 rm -rf /var/lib/apt/lists/*\"]},\"author\":\"NGINX Docker Maintainers \\\"docker-maint@nginx.com\\\"\"}"
      },
      {
         "v1Compatibility": "{\"id\":\"5cf74bcb1bde2e2249824a682f45235954543a5d57081db22c96402342db49e9\",\"parent\":\"699cf123e2fea6a56d00f0506b4f4f8005ffaf324bf2dbd4359f6d2596aa9a0c\",\"created\":\"2017-03-27T19:48:19.151777495Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop)  ENV NGINX_VERSION=1.10.3-1~jessie\"]},\"author\":\"NGINX Docker Maintainers \\\"docker-maint@nginx.com\\\"\",\"throwaway\":true}"
      },
      {
         "v1Compatibility": "{\"id\":\"699cf123e2fea6a56d00f0506b4f4f8005ffaf324bf2dbd4359f6d2596aa9a0c\",\"parent\":\"c35ece8820ad1a5c6828908fa194dcc007bd59d622b19e282ef7f4e857710b15\",\"created\":\"2017-03-21T22:06:58.207888376Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop)  MAINTAINER NGINX Docker Maintainers \\\"docker-maint@nginx.com\\\"\"]},\"author\":\"NGINX Docker Maintainers \\\"docker-maint@nginx.com\\\"\",\"throwaway\":true}"
      },
      {
         "v1Compatibility": "{\"id\":\"c35ece8820ad1a5c6828908fa194dcc007bd59d622b19e282ef7f4e857710b15\",\"parent\":\"c1f98057d627496e1843020c45c444f7572b581ef440e7c177f3a4834d483609\",\"created\":\"2017-03-21T18:29:05.091744235Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop)  CMD [\\\"/bin/bash\\\"]\"]},\"throwaway\":true}"
      },
      {
         "v1Compatibility": "{\"id\":\"c1f98057d627496e1843020c45c444f7572b581ef440e7c177f3a4834d483609\",\"created\":\"2017-03-21T18:28:51.055495122Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop) ADD file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in / \"]}}"
      }
   ],
   "signatures": [
      {
         "header": {
            "jwk": {
               "crv": "P-256",
               "kid": "CXE3:O6RL:2F2O:NQCB:KADW:52ZT:2RC5:V4SX:KRD3:CUNF:S5M2:ORN4",
               "kty": "EC",
               "x": "4dXF72StsoliZmhXaW_B9c7GyjiXfNnegHjgHj9yzIk",
               "y": "hRSdq4Tglw-xkcggRg2qd0cVW-OEbmGwYAIpOhUmM6M"
            },
            "alg": "ES256"
         },
         "signature": "exT6DH1vLONDRVX3w0am5jxv5vqOBrWwlrFUdIdLEpO9C4K407HSNr9kdO0wWwzEKAux-fjqbGmMOLMsQjbyQQ",
         "protected": "eyJmb3JtYXRMZW5ndGgiOjYyODEsImZvcm1hdFRhaWwiOiJDbjAiLCJ0aW1lIjoiMjAxNy0xMS0yN1QyMToxNToyN1oifQ"
      }
   ]
}`

func newMockRegistry() {
	masterRouter := http.NewServeMux()
	// Handle
	masterRouter.HandleFunc("/v2/library/nginx/manifests/1.10", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.Header.Get("Accept") != manifestV1.MediaTypeManifest {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, getMetadataPayload)
		} else if r.Method == "HEAD" {
			w.Header().Add("Docker-Content-Digest", "sha256:693e49e96066feacf922ff62bb87dfae3865cca7621bdf22afd053bf1c0cc37d")
		}
	})
	masterRouter.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{}")
	})
	server = httptest.NewServer(masterRouter)
}

func getProtoTimestamp(nanos int64) *timestamp.Timestamp {
	proto, _ := ptypes.TimestampProto(time.Unix(0, nanos))
	return proto
}

func TestGetMetadata(t *testing.T) {
	newMockRegistry()

	dockerHubClient, err := newRegistry(&v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Docker{
			Docker: &v1.DockerConfig{
				Endpoint: "http://" + server.Listener.Addr().String(),
			},
		},
	})
	require.NoError(t, err)

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
		Author:      author,
		Created:     getProtoTimestamp(1490644114817993292),
		RegistrySha: "sha256:693e49e96066feacf922ff62bb87dfae3865cca7621bdf22afd053bf1c0cc37d",
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
		FsLayers: []string{
			"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4",
			"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4",
			"sha256:556c62bb43ac9073f4dfc95383e83f8048633a041cb9e7eb2c1f346ba39a5183",
			"sha256:1e3e18a64ea9924fd9688d125c2844c4df144e41b1d2880a06423bca925b778c",
			"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4",
			"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4",
			"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4",
			"sha256:6d827a3ef358f4fa21ef8251f95492e667da826653fd43641cef5a877dc03a70",
		},
	}
	assert.Equal(t, expectedMetadata, metadata)
}
