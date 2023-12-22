package rhel

import (
	"strconv"
	"sync"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types/cpe"
)

// RelMap memoizes the Distributions handed out by this package.
//
// Doing this is a cop-out to the previous approach of having a hard-coded set of structs.
// In the case something is (mistakenly) doing pointer comparisons, this will make that work
// but still allows us to have the list of distributions grow ad-hoc.
var relMap sync.Map

func mkRelease(n int64) *claircore.Distribution {
	v, ok := relMap.Load(n)
	if !ok {
		s := strconv.FormatInt(n, 10)
		v, _ = relMap.LoadOrStore(n, &claircore.Distribution{
			Name:       "Red Hat Enterprise Linux Server",
			Version:    s,
			VersionID:  s,
			DID:        "rhel",
			PrettyName: "Red Hat Enterprise Linux Server " + s,
			CPE:        cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:" + s),
		})
	}
	return v.(*claircore.Distribution)
}
