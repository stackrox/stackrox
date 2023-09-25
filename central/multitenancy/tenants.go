package multitenancy

import "github.com/stackrox/rox/pkg/logging"

var user string

func SetUser(u string) {
	user = u
	logging.Logger().Infof("USER %s, TENANTID: %d", u, GetTenantID())
}

func GetTenantID() int {
	return Tenants[user]
}

var Tenants = map[string]int{
	"admin":       1,
	"littleadmin": 1,
	"simon":       2,
	"kyle":        3,
}
