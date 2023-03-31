package fake

import (
	"bytes"

	"github.com/cockroachdb/pebble"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	idSeparator = "\x00"

	namespacePrefix      = "namespaces"
	podPrefix            = "pods"
	deploymentPrefix     = "deployments"
	replicaSetPrefix     = "replicasets"
	nodePrefix           = "nodes"
	serviceAccountPrefix = "serviceaccount"
	rolesPrefix          = "roles"
	rolebindingsPrefix   = "rolebindings"
	servicePrefix        = "services"
	networkPolicyPrefix  = "networkpolicies"
)

func (w *WorkloadManager) writeID(prefix, id string) error {
	return w.db.Set([]byte(prefix+idSeparator+id), []byte{}, &pebble.WriteOptions{})
}

func (w *WorkloadManager) getIDsForPrefix(prefix string) []string {
	it := w.db.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
	})
	var ids []string
	for it.Next() {
		if err := it.Error(); err != nil {
			log.Panicf("error iterating: %v", err)
		}
		if !bytes.HasPrefix(it.Key(), []byte(prefix)) {
			break
		}
		ids = append(ids, stringutils.GetAfter(string(it.Key()), idSeparator))
	}
	if err := it.Close(); err != nil {
		log.Panicf("could not close iterator: %v", err)
	}
	return ids
}
