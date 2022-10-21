# Process Listening On Port

This structure represent what it says, namely observed processes that are
listening on a certain port. It's being represented via two structures
throghout the implementation: `ProcessListeningOnPort` for the API (internal
and user facing) purposes and `ProcessListeningOnPortStorage` for actually
storing it in the database. The difference between two is that the former
contains necessary process information embedded into it via
`ProcessIndicatorUniqueKey`, while the latter has only a foreign key to a
corresponding record in the `process_indicators` table.

Due to the various requirements the storage behind
`ProcessListeningOnPortStorage` contains manually written bits to support
queries with joins. Note that for the efficient use of this structure a
corresponding index has to be defined on `ProcessIndicator`. This also means
that the PLOP will still incur some overhead even when the feature flag is
disabled.
