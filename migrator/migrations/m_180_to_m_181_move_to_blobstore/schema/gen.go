package schema

// TODO(ROX-17180): Remove this auto-generation at the beginning of 4.2 or at least
// before we made schema change to Blob store after first release.

//go:generate pg-schema-migration-helper --type=storage.Blob
