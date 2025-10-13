package resources

import (
	"k8s.io/cli-runtime/pkg/resource"
)

// GetK8sInfos reads location on disk and returns k8s objects with warnings and errors
func GetK8sInfos(path string, failFast bool, treatWarningsAsErrors bool) (nfos []*resource.Info, warns []error, errs []error) {
	infos, err := getK8sInfos(path, failFast, treatWarningsAsErrors)
	warns, errs = handleAggregatedError(err)
	return infos, warns, errs
}

func getK8sInfos(path string, failFast bool, treatWarningsAsErrors bool) ([]*resource.Info, error) {
	b := resource.NewLocalBuilder().
		Unstructured()
	if !(failFast && treatWarningsAsErrors) {
		b.ContinueOnError()
	}
	//nolint:wrapcheck // we do wrap the errors later in ErrorHandler
	return b.Path(true, path).Flatten().Do().Infos()
}
