package dtr

import "time"

// Replica is one instance of DTR
type Replica struct {
	DBUpdatedAt time.Time `json:"db_updated_at"`
	Version     string    `json:"version"`
	ReplicaID   string    `json:"replica_id"`
}

// scannerMetadata is the metadata returned from getting the DTR server status
type scannerMetadata struct {
	State              int                `json:"state"`
	ScannerVersion     int                `json:"scanner_version"`
	ScannerUpdatedAt   time.Time          `json:"scanner_updated_at"`
	DBVersion          int                `json:"db_version"`
	DBUpdatedAt        time.Time          `json:"db_updated_at"`
	LastDBUpdateFailed bool               `json:"last_db_update_failed"`
	Replicas           map[string]Replica `json:"replicas"`
}

type metadataFeatures struct {
	ScanningEnabled   bool   `json:"scanningEnabled"`
	ScanningLicensed  bool   `json:"scanningLicensed"`
	PromotionLicensed bool   `json:"promotionLicensed"`
	DBVersion         int    `json:"db_version"`
	UCPHost           string `json:"ucpHost"`
}
