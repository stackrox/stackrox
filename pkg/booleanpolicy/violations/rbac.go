package violations

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/search"
)

var (
	permissionToDescMap = map[string]string{
		"NONE":                  "no specified access",
		"DEFAULT":               "default access",
		"ELEVATED_IN_NAMESPACE": "elevated access in namespace",
		"ELEVATED_CLUSTER_WIDE": "elevated access cluster wide",
		"CLUSTER_ADMIN":         "cluster admin access"}
)

func rbacPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	msgTemplate := "Service account permission level with %s"
	permissionLevel, err := getSingleValueFromFieldMap(search.ServiceAccountPermissionLevel.String(), fieldMap)
	if err != nil || permissionLevel == "" {
		return nil, errors.New("missing permission level")
	}
	permissionDesc, ok := permissionToDescMap[strings.ToUpper(permissionLevel)]
	if !ok {
		return nil, errors.New("unexpected permission level")
	}
	return []string{fmt.Sprintf(msgTemplate, permissionDesc)}, nil
}
