package violations

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/search"
)

var (
	portExposureToDescMap = map[string]string{
		"EXTERNAL": "exposed with load balancer",
		"NODE":     "exposed on node port",
		"HOST":     "exposed on host port",
		"INTERNAL": "using internal cluster IP"}
)

func portExposurePrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	msgTemplate := "Deployment port(s) %s"
	exposureLevel, err := getSingleValueFromFieldMap(search.ExposureLevel.String(), fieldMap)
	if err != nil || exposureLevel == "" {
		return nil, errors.New("missing port exposure level")
	}
	portExposureDesc, ok := portExposureToDescMap[strings.ToUpper(exposureLevel)]
	if !ok {
		return nil, errors.New("unexpected port exposure level")
	}
	return []string{fmt.Sprintf(msgTemplate, portExposureDesc)}, nil
}
