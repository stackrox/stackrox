package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extraDiscoveryInfo_SelectResponseMode(t *testing.T) {
	tests := []struct {
		name            string
		info            *extraDiscoveryInfo
		hasClientSecret bool
		want            string
		wantErr         bool
	}{
		// -------------------------------- With client secret
		{
			name: "error with secret",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"code"},
				ResponseModesSupported: []string{"alien"},
			},
			hasClientSecret: true,
			want:            "",
			wantErr:         true,
		},
		{
			name: "post with code",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"code"},
				ResponseModesSupported: []string{"form_post", "fragment", "query"},
			},
			hasClientSecret: true,
			want:            "form_post",
			wantErr:         false,
		},
		{
			name: "post without code",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"id_token token"},
				ResponseModesSupported: []string{"form_post", "fragment", "query"},
			},
			hasClientSecret: true,
			want:            "form_post",
			wantErr:         false,
		},
		{
			name: "no post with code",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"code"},
				ResponseModesSupported: []string{"fragment", "query"},
			},
			hasClientSecret: true,
			want:            "query",
			wantErr:         false,
		},
		{
			name: "no post and without code",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"id_token token"},
				ResponseModesSupported: []string{"fragment", "query"},
			},
			hasClientSecret: true,
			want:            "fragment",
			wantErr:         false,
		},
		{
			name: "google with secret",
			info: &extraDiscoveryInfo{
				ResponseModesSupported: nil,
			},
			hasClientSecret: true,
			want:            "form_post",
			wantErr:         false,
		},
		// ----------------------------------------- without client secret
		{
			name: "error without secret",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"code"},
				ResponseModesSupported: []string{"alien"},
			},
			hasClientSecret: false,
			want:            "",
			wantErr:         true,
		},
		{
			name: "post with code, no secret",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"code"},
				ResponseModesSupported: []string{"form_post", "fragment", "query"},
			},
			hasClientSecret: false,
			want:            "form_post",
			wantErr:         false,
		},
		{
			name: "post without code, no secret",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"id_token token"},
				ResponseModesSupported: []string{"form_post", "fragment", "query"},
			},
			hasClientSecret: false,
			want:            "form_post",
			wantErr:         false,
		},
		{
			name: "no post with code, no secret",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"code"},
				ResponseModesSupported: []string{"fragment", "query"},
			},
			hasClientSecret: false,
			want:            "fragment",
			wantErr:         false,
		},
		{
			name: "no post and without code, no secret",
			info: &extraDiscoveryInfo{
				ResponseTypesSupported: []string{"id_token token"},
				ResponseModesSupported: []string{"fragment", "query"},
			},
			hasClientSecret: false,
			want:            "fragment",
			wantErr:         false,
		},
		{
			name: "google without secret",
			info: &extraDiscoveryInfo{
				ResponseModesSupported: nil,
			},
			hasClientSecret: false,
			want:            "form_post",
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.info.SelectResponseMode(tt.hasClientSecret)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectResponseMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SelectResponseMode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extraDiscoveryInfo_SelectResponseType(t *testing.T) {
	tests := map[string]struct {
		supportedResponseTypes []string
		wantWithSecret         map[string]string
		wantNoSecret           map[string]string
	}{
		// List of providers based primarily on https://stack-rox.atlassian.net/wiki/spaces/~k/pages/709689387/Test+SSO+Integrations
		"sr-dev.auth0.com": {
			supportedResponseTypes: []string{"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token"},
			// note: PingFederate exposes exactly the same list, see https://stackrox.zendesk.com/agent/tickets/936
			wantWithSecret: map[string]string{
				"form_post": "code",
				"query":     "code",
				"fragment":  "token id_token",
			},
			wantNoSecret: map[string]string{
				"form_post": "token id_token",
				"query":     "", // error
				"fragment":  "token id_token",
			},
		},
		"gitlab.com": {
			supportedResponseTypes: []string{"code", "token"},
			wantWithSecret: map[string]string{
				"query":    "code",
				"fragment": "token",
			},
			wantNoSecret: map[string]string{
				"query":    "", // error
				"fragment": "token",
			},
		},
		"accounts.google.com": {
			supportedResponseTypes: []string{"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token", "none"},
			// note: KeyCloak exposes effectively the same list, see https://stackrox.zendesk.com/agent/tickets/954
			wantWithSecret: map[string]string{
				"form_post": "code",
				"query":     "code",
				"fragment":  "token id_token",
			},
			wantNoSecret: map[string]string{
				"form_post": "token id_token",
				"query":     "", // error
				"fragment":  "token id_token",
			},
		},
		"stackrox.oktapreview.com": {
			supportedResponseTypes: []string{"code", "id_token", "code id_token", "code token", "id_token token", "code id_token token"},
			wantWithSecret: map[string]string{
				"form_post": "code",
				"query":     "code",
				"fragment":  "id_token token",
			},
			wantNoSecret: map[string]string{
				"form_post": "id_token token",
				"query":     "", // error
				"fragment":  "id_token token",
			},
		},
		"hypothetical hybrid-but-no-pure-code provider": {
			supportedResponseTypes: []string{"token", "id_token", "code token", "code id_token", "token id_token", "code token id_token"},
			wantWithSecret: map[string]string{
				"form_post": "code token id_token",
				"query":     "", // error
				"fragment":  "token id_token",
			},
			wantNoSecret: map[string]string{
				"form_post": "token id_token",
				"query":     "", // error
				"fragment":  "token id_token",
			},
		},
		"error": {
			supportedResponseTypes: []string{"none"},
			wantWithSecret: map[string]string{
				"form_post": "", // error
			},
			wantNoSecret: map[string]string{
				"form_post": "", // error
			},
		},
	}
	secretStrings := map[bool]string{
		true:  "-with-secret",
		false: "-no-secret",
	}
	for name, tt := range tests {
		for _, hasClientSecret := range []bool{true, false} {
			var wantMap map[string]string
			if hasClientSecret {
				wantMap = tt.wantWithSecret
			} else {
				wantMap = tt.wantNoSecret
			}
			for mode := range wantMap {
				t.Run(name+"-"+mode+secretStrings[hasClientSecret], func(t *testing.T) {
					info := extraDiscoveryInfo{ResponseTypesSupported: tt.supportedResponseTypes}
					got, err := info.SelectResponseType(mode, hasClientSecret)
					want := wantMap[mode]
					if want == "" {
						assert.Error(t, err)
					} else {
						assert.NoError(t, err)
						assert.Equal(t, want, got)
					}
				})
			}
		}
	}
}

func Test_extraDiscoveryInfo_SupportsResponseMode(t *testing.T) {
	tests := []struct {
		name          string
		responseModes []string
		responseMode  string
		want          bool
	}{
		{
			name:          "empty",
			responseModes: []string{},
			responseMode:  "whatever",
			want:          false,
		},
		{
			name:          "unknown modes",
			responseModes: nil,
			responseMode:  "whatever",
			want:          true,
		},
		{
			name:          "missing",
			responseModes: []string{"something", "else"},
			responseMode:  "foo",
			want:          false,
		},
		{
			name:          "present",
			responseModes: []string{"one", "two", "three"},
			responseMode:  "two",
			want:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &extraDiscoveryInfo{
				ResponseModesSupported: tt.responseModes,
			}
			if got := i.SupportsResponseMode(tt.responseMode); got != tt.want {
				t.Errorf("SupportsResponseMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extraDiscoveryInfo_SupportsResponseType(t *testing.T) {
	tests := []struct {
		name          string
		responseTypes []string
		responseType  string
		want          bool
	}{
		{
			name:          "empty",
			responseTypes: []string{},
			responseType:  "whatever",
			want:          false,
		},
		{
			name:          "unknown",
			responseTypes: nil,
			responseType:  "whatever",
			want:          false,
		},
		{
			name:          "missing",
			responseTypes: []string{"something", "else"},
			responseType:  "foo",
			want:          false,
		},
		{
			name:          "present",
			responseTypes: []string{"one", "two", "three"},
			responseType:  "two",
			want:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &extraDiscoveryInfo{
				ResponseTypesSupported: tt.responseTypes,
			}
			if got := i.SupportsResponseType(tt.responseType); got != tt.want {
				t.Errorf("SupportsResponseType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extraDiscoveryInfo_SupportsScope(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		scope  string
		want   bool
	}{
		{
			name:   "empty",
			scopes: []string{},
			scope:  "whatever",
			want:   false,
		},
		{
			name:   "unknown",
			scopes: nil,
			scope:  "whatever",
			want:   false,
		},
		{
			name:   "missing",
			scopes: []string{"something", "else"},
			scope:  "foo",
			want:   false,
		},
		{
			name:   "present",
			scopes: []string{"one", "two", "three"},
			scope:  "two",
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &extraDiscoveryInfo{
				ScopesSupported: tt.scopes,
			}
			if got := i.SupportsScope(tt.scope); got != tt.want {
				t.Errorf("SupportsScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_selectPreferred(t *testing.T) {
	type args struct {
		options     []string
		preferences []string
	}
	tests := []struct {
		name   string
		args   args
		want   string
		wantOK bool
	}{
		{
			name:   "empty",
			want:   "",
			wantOK: false,
		},
		{
			name: "no options",
			args: args{
				options:     nil,
				preferences: []string{"foo"},
			},
			want:   "",
			wantOK: false,
		},
		{
			name: "no preferences",
			args: args{
				options:     []string{"foo"},
				preferences: nil,
			},
			want:   "",
			wantOK: false,
		},
		{
			name: "no overlap",
			args: args{
				options:     []string{"a"},
				preferences: []string{"b"},
			},
			want:   "",
			wantOK: false,
		},
		{
			name: "first preference available",
			args: args{
				options:     []string{"a", "b", "c"},
				preferences: []string{"d", "c", "b"},
			},
			want:   "c",
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOK := selectPreferred(tt.args.options, tt.args.preferences)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOK, gotOK)
		})
	}
}
