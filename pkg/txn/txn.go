package txn

import "github.com/stackrox/stackrox/pkg/logging"

var log = logging.LoggerForModule()

// ReconciliationNeeded determines if based on the tx numbers reconciliation is necessary
// for the indexer and DB
func ReconciliationNeeded(dbTxNum, indexTxNum uint64) bool {
	return dbTxNum == 0 || indexTxNum != dbTxNum
}
