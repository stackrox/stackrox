package features

// StoreEventHashes stores the hashes of successfully processed objects we receive from Sensor into the database
var StoreEventHashes = registerFeature("Store Event Hashes", "ROX_STORE_EVENT_HASHES", enabled, unchangeableInProd)
