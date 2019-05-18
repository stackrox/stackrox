package store

//go:generate boltbindings-wrapper --object=StoredLicenseKey --singular=LicenseKey --bucket=licenseKeys --id-func=GetLicenseId --methods=list,upsert_many,delete
//go:generate mockgen-wrapper Store
