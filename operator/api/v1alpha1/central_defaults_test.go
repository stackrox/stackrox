package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
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
						CreateSCCs: ptr.To(true),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
			},
		},
		"explicit true wins": {
			before: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
					},
				},
			},
		},
		"explicit false wins": {
			before: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
			},
		},
		"defaulting true works": {
			before: &Central{
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
			},
		},
		"defaulting false works": {
			before: &Central{
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
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
						CreateSCCs: ptr.To(false),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(false),
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
						CreateSCCs: ptr.To(true),
					},
				},
			},
			after: &Central{
				Spec: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
					},
				},
				Defaults: CentralSpec{
					Misc: &MiscSpec{
						CreateSCCs: ptr.To(true),
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
