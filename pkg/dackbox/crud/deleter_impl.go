package crud

import (
	"github.com/stackrox/rox/pkg/dackbox"
)

type deleterImpl struct {
	removeFromIndex bool
	shared          bool
	partials        []PartialDeleter
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
	// Collect the keys the item currently points to in case we need to clean them up.
	partialKeys := g.GetRefsFrom(key)

	// Remove the key from the id map and the DB.
	g.DeleteRefsFrom(key)
	g.DeleteRefsTo(key)
	dackTxn.Delete(key)

	// Delete the partial objects. This needs to come after the shared check so that we can clean objects up in line.
	for _, partial := range dc.partials {
		if err := partial.DeletePartialsIn(partialKeys, dackTxn); err != nil {
			return err
		}
	}
	return nil
}

type partialDeleterImpl struct {
	matchFunction KeyMatchFunction

	deleter Deleter
}

// DeleteIn deletes the data for the input key on the input transaction.
func (dc *partialDeleterImpl) DeletePartialsIn(partialKeys [][]byte, dackTxn *dackbox.Transaction) error {
	// Get the currently stored dependent keys.
	for _, partialKey := range partialKeys {
		if dc.matchFunction != nil && !dc.matchFunction(partialKey) {
			continue
		}
		if err := dc.deleter.DeleteIn(partialKey, dackTxn); err != nil {
			return err
		}
	}
	return nil
}
