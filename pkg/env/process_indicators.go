package env

var (
	// ProcessAddBatchSize defines the number of process indicators to write in a single batch
	ProcessAddBatchSize = RegisterIntegerSetting("ROX_CENTRAL_ADD_PROCESS_BATCH_SIZE", 5000)
)
