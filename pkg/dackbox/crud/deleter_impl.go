package crud

import (
	"github.com/stackrox/stackrox/pkg/dackbox"
)

type deleterImpl struct {
	removeFromIndex bool
	shared          bool
}

// DeleteIn deletes the data for the input key on the input transaction.
func (dc *deleterImpl) DeleteIn(key []byte, dackTxn *dackbox.Transaction) error {
	g := dackTxn.Graph()
	// If shared, check that no more references exist for the object before deleting.
	if dc.shared {
		if g.CountRefsTo(key) > 0 {
			return nil
		}
	}
	// If indexed, add the key to the set of dirty keys.
	if dc.removeFromIndex {
		dackTxn.MarkDirty(key, nil)
	}

	// Remove the key from the id map and the DB.
	g.DeleteRefsFrom(key)
	g.DeleteRefsTo(key)
	dackTxn.Delete(key)

	return nil
}
