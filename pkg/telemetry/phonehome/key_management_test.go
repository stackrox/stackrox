package phonehome

import (
	"testing"
)

func Test_toDownload(t *testing.T) {
	tests := map[string]struct {
		release bool
		key     string
		cfgURL  string

		download bool
	}{
		"a": {release: false, key: "", cfgURL: "", download: false},
		"b": {release: false, key: "abc", cfgURL: "", download: false},
		"c": {release: false, key: "", cfgURL: "url", download: false},
		"d": {release: false, key: "abc", cfgURL: "url", download: true},

		"e": {release: true, key: "", cfgURL: "", download: false},
		"f": {release: true, key: "abc", cfgURL: "", download: false},
		"g": {release: true, key: "", cfgURL: "url", download: true},
		"h": {release: true, key: "abc", cfgURL: "url", download: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := toDownload(tt.release, tt.key, tt.cfgURL); got != tt.download {
				t.Errorf("toDownload() = %v, want %v", got, tt.download)
			}
		})
	}
}

func Test_useRemoteKey(t *testing.T) {
	tests := map[string]struct {
		release  bool
		cfg      *remoteConfig
		localKey string

		useRemote bool
	}{
		"a": {release: false, cfg: nil, localKey: "", useRemote: false},
		"b": {release: false, cfg: &remoteConfig{Key: "abc"}, localKey: "", useRemote: false},
		"c": {release: false, cfg: nil, localKey: "", useRemote: false},
		"d": {release: false, cfg: &remoteConfig{Key: "abc"}, localKey: "abc", useRemote: true},
		"e": {release: false, cfg: &remoteConfig{Key: "abc"}, localKey: "def", useRemote: false},

		"f": {release: true, cfg: nil, localKey: "", useRemote: false},
		"g": {release: true, cfg: &remoteConfig{Key: "abc"}, localKey: "", useRemote: true},
		"h": {release: true, cfg: nil, localKey: "", useRemote: false},
		"i": {release: true, cfg: &remoteConfig{Key: "abc"}, localKey: "abc", useRemote: true},
		"j": {release: true, cfg: &remoteConfig{Key: "abc"}, localKey: "def", useRemote: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := useRemoteKey(tt.release, tt.cfg, tt.localKey); got != tt.useRemote {
				t.Errorf("useRemoteKey() = %v, want %v", got, tt.useRemote)
			}
		})
	}
}
