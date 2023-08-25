package fake

import (
	"bytes"

	"github.com/cockroachdb/pebble"
	"github.com/stackrox/rox/pkg/stringutils"
	"k8s.io/apimachinery/pkg/types"
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

func idOrNewUID(id string) types.UID {
	if id != "" {
		return types.UID(id)
	}
	return newUUID()
}

func getID(ids []string, idx int) string {
	if len(ids) <= idx {
		return ""
	}
	return ids[idx]
}

func getPebbleKey(prefix, id string) []byte {
	return []byte(prefix + idSeparator + id)
}

func (w *WorkloadManager) writeID(prefix string, uid types.UID) {
	if w.db == nil {
		return
	}
	if err := w.db.Set(getPebbleKey(prefix, string(uid)), []byte{}, pebble.Sync); err != nil {
		log.Errorf("writing id: %s %s", prefix, uid)
	}
}

func (w *WorkloadManager) deleteID(prefix string, uid types.UID) {
	if w.db == nil {
		return
	}
	if err := w.db.Delete(getPebbleKey(prefix, string(uid)), pebble.Sync); err != nil {
		log.Errorf("deleting id: %s %s", prefix, uid)
	}
}

func (w *WorkloadManager) getIDsForPrefix(prefix string) []string {
	if w.db == nil {
		return nil
	}
	it := w.db.NewIter(&pebble.IterOptions{})

	prefixKey := getPebbleKey(prefix, "")
	var ids []string
	for it.SeekGE(prefixKey); it.Valid() && bytes.HasPrefix(it.Key(), prefixKey); it.Next() {
		ids = append(ids, stringutils.GetAfter(string(it.Key()), idSeparator))
	}
	if err := it.Error(); err != nil {
		log.Panicf("error in iterator: %v", err)
	}
	if err := it.Close(); err != nil {
		log.Panicf("error closing iterator: %v", err)
	}
	log.Infof("%s: %+v", prefix, len(ids))
	return ids
}
