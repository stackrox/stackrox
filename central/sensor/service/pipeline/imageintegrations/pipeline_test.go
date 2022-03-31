package imageintegrations

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestParseEndpoint(t *testing.T) {
	cases := []struct {
		endpoint string

		url string
	}{
		{
			endpoint: "https://docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "index.docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "registry-1.docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "https://registry-1.docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "https://index.docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "https://index.docker.io/v1",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "https://myregistry.hello.io",
			url:      "https://myregistry.hello.io",
		},
		{
			endpoint: "myregistry.hello.io",
			url:      "https://myregistry.hello.io",
		},
		{
			endpoint: "myregistry.hello.io/v1/randompage",
			url:      "https://myregistry.hello.io",
		},
		{
			endpoint: "http://myregistry.hello.io/v1/randompage",
			url:      "http://myregistry.hello.io",
		},
		{
			endpoint: "http://myregistry.hello.io:5000/v1/randompage",
			url:      "http://myregistry.hello.io:5000",
		},
	}
	for _, c := range cases {
		t.Run(c.endpoint, func(t *testing.T) {
			url := parseEndpointForURL(c.endpoint)
			assert.Equal(t, c.url, url)
		})
	}
}

func Test_matchesECRAuth(t *testing.T) {
	createIntegration := func(authData *storage.ECRConfig_AuthorizationData) *storage.ImageIntegration {
		return &storage.ImageIntegration{
			IntegrationConfig: &storage.ImageIntegration_Ecr{
				Ecr: &storage.ECRConfig{
					AuthorizationData: authData,
				},
			},
		}
	}
	createAuthData := func(u, p string, e time.Time) *storage.ECRConfig_AuthorizationData {
		ts, err := types.TimestampProto(e)
		if err != nil {
			assert.FailNow(t, "failed to convert timestamp: %v", err)
		}
		return &storage.ECRConfig_AuthorizationData{
			Username:  u,
			Password:  p,
			ExpiresAt: ts,
		}
	}
	type args struct {
		this  *storage.ImageIntegration
		other *storage.ImageIntegration
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should match on nil",
			args: args{
				this:  createIntegration(nil),
				other: createIntegration(nil),
			},
			want: true,
		},
		{
			name: "should match on same username, password and expire at",
			args: args{
				this:  createIntegration(createAuthData("foo", "bar", time.Unix(0, 0))),
				other: createIntegration(createAuthData("foo", "bar", time.Unix(0, 0))),
			},
			want: true,
		},
		{
			name: "should not match on different username",
			args: args{
				this:  createIntegration(createAuthData("foo", "bar", time.Unix(0, 0))),
				other: createIntegration(createAuthData("otherfoo", "bar", time.Unix(0, 0))),
			},
		},
		{
			name: "should not match on different password",
			args: args{
				this:  createIntegration(createAuthData("foo", "bar", time.Unix(0, 0))),
				other: createIntegration(createAuthData("foo", "password", time.Unix(0, 0))),
			},
		},
		{
			name: "should not match on different expired",
			args: args{
				this:  createIntegration(createAuthData("foo", "bar", time.Unix(0, 0))),
				other: createIntegration(createAuthData("foo", "password", time.Unix(10, 10))),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, matchesECRAuth(tt.args.this, tt.args.other), "matchesECRAuth(%v, %v)", tt.args.this, tt.args.other)
		})
	}
}
