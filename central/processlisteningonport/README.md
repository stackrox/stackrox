# Process Listening On Port

This structure represent what it says, namely observed processes that are
listening on a certain port. Logically it's implemented as a data structure
that contains reference to a corresponding ProcessIndicator, which is transformed
into an FK on the database level. PLOP is represented via three structures
throughout the implementation: `ProcessListeningOnPort` for the user facing API
purposes, `ProcessListeningOnPortFromSensor` for the internal API and
`ProcessListeningOnPortStorage` for actually storing it in the database.
The difference between `ProcessListeningOnPortStorage` and
`ProcessListeningOnPortFromSensor` is that the former contains a foreign key to
a corresponding record in the `process_indicators`. It may or may not contain
process information depending upon if the `process_indicators` table contains
matching id and therefore corresponding process information.

`ProcessListeningOnPort` contains additional information obtained from the 
`process_indicators` table by joining on processindicatorsid from the
process_listening_on_ports with the id from the process_indicators table.

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

PLOP lifecycle looks like this:

* When the process starts listen on a port, Collector sends the PLOP event with
  CloseTimestamp = null to indicate that the endpoint is active.
* As soon as the process finishes listening, Collector sends a new PLOP event
  with the same information and CloseTimestamp set to an actual timestamp
  value. This indicates that the PLOP object is closed, have to be excluded
  from the API (which only returns active endpoints) and will be cleaned up
  together with the process during process pruning.

In case if you need to troubleshoot PLOP, there are following metrics
available (including generated metrics):

Storage:

* ProcessListeningOnPortStorage.UpsertMany -- represents the time frame
  how long does it take to add or replace new PLOP objects

* ProcessListeningOnPortStorage.RemoveMany -- how long does it take to remove
  PLOP objects (being used for pruning with process information)

* ProcessListeningOnPortStorage.GetByQuery --  time it takes to fetch PLOPs
  (used to fetch esiting record for marking them as closed)

* ProcessListeningOnPortStorage.GetProcessListeningOnPort -- represents how
  long does it take to fetch PLOP objects for the API

There are more storage metrics available, but operations they represent are not
in use at the moment.

Internal API:

* ProcessListeningOnPort -- counter, represents how many PLOP objects Sensor
  has sent to the Central.
