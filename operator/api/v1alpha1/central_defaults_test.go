package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeCentralDefaultsIntoSpec(t *testing.T) {
	tests := map[string]struct {
		before *Central
		after  *Central
	}{
		"empty": {
			before: &Central{},
			after:  &Central{},
		},
		"untouched": {
			before: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
			},
		},
		"explicit true wins": {
			before: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
			},
		},
		"explicit false wins": {
			before: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
			},
		},
		"defaulting true works": {
			before: &Central{
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
			},
		},
		"defaulting false works": {
			before: &Central{
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
			},
		},
		"defaulting false into empty struct works": {
			before: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(false),
					},
				},
			},
		},
		"defaulting true into empty struct works": {
			before: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: new(true),
					},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			central := tt.before.DeepCopy()
			require.NoError(t, MergeCentralDefaultsIntoSpec(central))
			require.Equal(t, tt.after, central)
		})
	}
}
