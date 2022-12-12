# Process Listening On Port

This structure represent what it says, namely observed processes that are
listening on a certain port. Logically it's implemented as a data structure
that contain reference to a corresponding ProcessIndicator, what is transformed
into an FK on the database level. PLOP is being represented via two structures
throughout the implementation: `ProcessListeningOnPort` for the API (internal
and user facing) purposes and `ProcessListeningOnPortStorage` for actually
storing it in the database. The difference between two is that the former
contains necessary process information embedded into it via
`ProcessIndicatorUniqueKey`, while the latter has only a foreign key to a
corresponding record in the `process_indicators` table.

Due to the various requirements the storage behind
`ProcessListeningOnPortStorage` contains manually written bits to support
queries with joins. Note that for the efficient use of this structure a
corresponding index has to be defined on `ProcessIndicator`.

The implementation of data storage for PLOP objects is not very restrictive,
because there are certain cases where an assumption "one port listener" - "one
process indicator" could be wrong, e.g. when multiple processes listening on
the same port via SO_REUSE_PORT, or multiple processes are the same up to the
executable file path, process name and arguments. In such situations it could
happen that a PLOP object will be stored without a corresponding
ProcessIndicator reference to not lose the data and facilitate future
troubleshooting.

From the Scope Access Control perspective, ProcessListeningOnPort is falling
into the DeploymentExtension category and being managed accordingly.

ProcessListeningOnPort is included into the process pruning as an additional
cleaning step, i.e. before actually deleting orphaned ProcessIndicators all the
PLOP objects referencing to-be-deleted Indicators are going to be removed as
well.

PLOP object is not implemented as an extension of currently existing Process
information nor Networking information, because:

* Networking information is specified per deployment, PLOP is per container.
* Adding PLOP directly into Process Indicator will be perilous to the
  performance of queries against it.
* Making PLOP a separate entity we ensure flexibility from which side we
  collect information: either it's going to be from the process (then join the
  process to the listening info) or from the networking point of view (join
  listening to the process info).
