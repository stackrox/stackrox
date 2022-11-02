package framework

import "helm.sh/helm/v3/pkg/chartutil"

// apply modifies a `ReleaseOptions` object according to the overrides specified in the `ReleaseSpec` section (while
// leaving everything else untouched).
func (s *ReleaseSpec) apply(opts *chartutil.ReleaseOptions) {
	if s.Name != "" {
		opts.Name = s.Name
	}
	if s.Namespace != "" {
		opts.Namespace = s.Namespace
	}
	if s.Revision != nil {
		opts.Revision = *s.Revision
	}
	if s.IsInstall != nil {
		opts.IsInstall = *s.IsInstall
	}
	if s.IsUpgrade != nil {
		opts.IsUpgrade = *s.IsUpgrade
	}
}
