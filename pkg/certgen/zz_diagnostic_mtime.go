package certgen

import "time"

// zzDiagnosticMtimeTest exists only to verify that GOCACHE's module index
// correctly invalidates a changed import even when file size is unchanged.
// Disposable diagnostic file — never merge to master....
var zzDiagnosticMtimeTest = time.Since
