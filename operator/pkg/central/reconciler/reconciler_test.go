package reconciler

import (
	"testing"

	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddSelectorOptionIfNeeded(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "empty selector",
			selector: "",
			wantLen:  0,
			wantErr:  false,
		}, {
			name:     "non-empty selector",
			selector: "foo=bar",
			wantLen:  1,
			wantErr:  false,
		}, {
			name:     "invalid selector",
			selector: "!",
			wantLen:  0,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addSelectorOptionIfNeeded(tt.selector, []pkgReconciler.Option{})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}
