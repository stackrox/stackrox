package crud

import (
	"github.com/stackrox/rox/pkg/dackbox"
)

type deleterImpl struct {
	gCFunc KeyMatchFunction
}

// DeleteIn deletes the data for the input key on the input transaction.
func (dc deleterImpl) DeleteIn(key []byte, dackTxn *dackbox.Transaction) error {
	// Get the currently stored dependent keys.
	partialKeys := dackTxn.Graph().GetRefsFrom(key)

	// Remove the key from the id map and the DB.
	err := dackTxn.Graph().DeleteRefs(key)
	if err != nil {
		return err
	}
	err = dackTxn.BadgerTxn().Delete(key)
	if err != nil {
		return err
	}

	// Remove any dependent keys
	if dc.gCFunc == nil {
		return nil
	}
	for _, partialKey := range partialKeys {
		if !dc.gCFunc(partialKey) {
			continue
		}
		if dackTxn.Graph().CountRefsTo(partialKey) == 0 {
			// No need to go through the partial write config since we don't need the key function.
			if err := dc.DeleteIn(partialKey, dackTxn); err != nil {
				return err
			}
		}
	}
	return nil
}
