package multitenancy

import "github.com/stackrox/rox/pkg/logging"

var log = logging.CreateLogger(logging.CurrentModule(), 0)

var user string

func SetUser(u string) {
	user = u
	log.Infof("USER %s, TENANTID: %d", u, GetTenantID())
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
