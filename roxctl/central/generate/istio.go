package generate

import (
	"strings"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/istioutils"
)

type istioSupportWrapper struct {
	istioSupport *string
}

func (w istioSupportWrapper) String() string {
	return *w.istioSupport
}

func (w istioSupportWrapper) Set(input string) error {
	_, err := istioutils.GetAPIResourcesByVersion(input)
	if err != nil {
		return errox.InvalidArgs.Newf("invalid Istio version %q. Valid versions are: %s (or leave empty "+
			"for no Istio support)", input, strings.Join(istioutils.ListKnownIstioVersions(), ", "))
	}
	*w.istioSupport = input
	return nil
}

func (w istioSupportWrapper) Type() string {
	return "version"
}
