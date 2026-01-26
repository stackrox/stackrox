package repositorytocpe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMappingFile_GetCPEs(t *testing.T) {
	tests := map[string]struct {
		mf       *MappingFile
		repoid   string
		wantCPEs []string
		wantOK   bool
	}{
		"existing repo returns CPEs": {
			mf: &MappingFile{
				Data: map[string]Repo{
					"rhel-8-server": {CPEs: []string{"cpe:/o:redhat:rhel:8"}},
				},
			},
			repoid:   "rhel-8-server",
			wantCPEs: []string{"cpe:/o:redhat:rhel:8"},
			wantOK:   true,
		},
		"missing repo returns false": {
			mf: &MappingFile{
				Data: map[string]Repo{
					"rhel-8-server": {CPEs: []string{"cpe:/o:redhat:rhel:8"}},
				},
			},
			repoid: "nonexistent",
			wantOK: false,
		},
		"nil receiver returns false": {
			mf:     nil,
			repoid: "anything",
			wantOK: false,
		},
		"empty data map returns false": {
			mf:     &MappingFile{Data: map[string]Repo{}},
			repoid: "anything",
			wantOK: false,
		},
		"multiple CPEs returned": {
			mf: &MappingFile{
				Data: map[string]Repo{
					"repo-1": {CPEs: []string{"cpe:1", "cpe:2", "cpe:3"}},
				},
			},
			repoid:   "repo-1",
			wantCPEs: []string{"cpe:1", "cpe:2", "cpe:3"},
			wantOK:   true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cpes, ok := tt.mf.GetCPEs(tt.repoid)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantCPEs, cpes)
		})
	}
}
