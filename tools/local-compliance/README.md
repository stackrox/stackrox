# Local Compliance

Local-compliance is a binary that behaves like the compliance component but runs locally.
It connects to Sensor running in the cluster and exchanges messages with it.

## Usage

Connecting to a running cluster and communicating with Sensor:

```bash
KUBECONFIG="~/.cluster/kubeconfig" ./tools/local-compliance/scripts/local-compliance.sh
```

## Details

Local-compliance utilises real production code of the compliance container.
It replaces its dependencies to decouple it from other components in a running cluster.
The dependencies are:
1. Logging - logger object to handle logs
2. NodeNameProvider - returns the name of the node on which the compliance is running.
   In real case this is taken from k8s, for local-compliance it is a hardcoded dummy value.
3. NodeScanner - responsible for communicating with the node-inventory container and obtaining node inventories.
   For local-compliance, it uses `loadGeneratingNodeScanner` that sends a fake node-inventory message every 100ms.
4. SensorReplyHandler - responsible for handling the ACK/NACK messages that Sensor sends back to Compliance.
   For local-compliance it prints a log line and does nothig.
