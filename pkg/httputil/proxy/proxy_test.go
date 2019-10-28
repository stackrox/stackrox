package proxy

import (
	"net/url"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestProxyConfig(t *testing.T) {
	mustParse := func(s string) *url.URL {
		ret, err := url.Parse(s)
		if err != nil {
			t.Error(err)
		}
		return ret
	}
	tests := []struct {
		name    string
		input   string
		want    *url.URL
		wantErr string
	}{
		{
			name:    "empty",
			input:   "",
			want:    nil,
			wantErr: "",
		},
		{
			name:    "simple",
			input:   `url: http://localhost:8080/`,
			want:    mustParse("http://localhost:8080/"),
			wantErr: "",
		},
		{
			name:    "user",
			input:   "url: http://localhost/\nusername: User",
			want:    mustParse("http://User@localhost/"),
			wantErr: "",
		},
		{
			name:    "userpass",
			input:   "url: http://localhost/\nusername: User\npassword: hunter2\n",
			want:    mustParse("http://User:hunter2@localhost/"),
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := proxyConfig{}
			err := yaml.Unmarshal([]byte(tt.input), &pc)
			if err != nil {
				t.Error(err)
			}
			got, err := pc.toURL()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
