package pulp

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestManifestLoad(t *testing.T) {
	unhex := func(s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	var want = Manifest{
		Entry{
			Path:     "RHEL8/openshift-4.1.oval.xml.bz2",
			Checksum: unhex("b067fe8942118b9dfa7ae24569601d1081e63b7050033953c9154324ff55cc27"),
			Size:     12994,
		},
		Entry{
			Path:     "RHEL8/openshift-4.2.oval.xml.bz2",
			Checksum: unhex("b067fe8942118b9dfa7ae24569601d1081e63b7050033953c9154324ff55cc27"),
			Size:     12994,
		},
		Entry{
			Path:     "RHEL8/openshift-4.3.oval.xml.bz2",
			Checksum: unhex("b067fe8942118b9dfa7ae24569601d1081e63b7050033953c9154324ff55cc27"),
			Size:     12994,
		},
		Entry{
			Path:     "RHEL8/openshift-4-including-unpatched.oval.xml.bz2",
			Checksum: unhex("040a8719cf7b2e5726cd96d642cd5fe6d24381cbd92f0caa99a977296b9070dc"),
			Size:     27469,
		},
		Entry{
			Path:     "RHEL8/openshift-4.oval.xml.bz2",
			Checksum: unhex("b067fe8942118b9dfa7ae24569601d1081e63b7050033953c9154324ff55cc27"),
			Size:     12994,
		},
		Entry{
			Path:     "RHEL8/openshift-service-mesh-1.0.oval.xml.bz2",
			Checksum: unhex("901b29d7928f55cf1b09928b75f2d6199ce8741c07a79bd010eca23d3fa85155"),
			Size:     5364,
		},
		Entry{
			Path:     "RHEL8/openshift-service-mesh-1.1.oval.xml.bz2",
			Checksum: unhex("6367a4bc26671e36bd0cd2bbad72c7f5a4b1f2a49948809684e586bc9233131d"),
			Size:     3340,
		},
	}
	t.Parallel()
	f, err := os.Open("testdata/PULP_MANIFEST")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var got Manifest
	if err := got.Load(f); err != nil {
		t.Error(err)
	}
	if !cmp.Equal(got, want) {
		t.Error(cmp.Diff(got, want))
	}
}
