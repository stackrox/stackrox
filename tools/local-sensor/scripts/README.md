# Local-sensor

Local Sensor is a binary entrypoint for Sensor to run it outside of a Kubernetes cluster. This can be helpful for development and debugging. Local sensor can be run in two modes:

- **Fake Central**: No Central installation needed. Messages can be output to terminal or a file.
- **Connected**: Connects to a real Central installation.

## Build local-sensor

```bash
go build -o local-sensor ./tools/local-sensor/
```

## Running local-sensor

To run local-sensor using **Fake Central** and writing Sensor Events to `local_sensor_output.json` file:

```bash
./local-sensor -central-out ./local_sensor_output.json
```

To run local-sensor against a real cluster, make sure you first have a cluster with ACS installed.

**(!)**: This method only works for secured clusters installed using Helm Charts.

```bash
# Scale down sensor running on the cluster
kubectl -n stackrox scale deployment sensor --replicas 0

# Fetch authentication certificates (this will store all files in ./tmp folder)
./tools/local-sensor/scripts/fetch-certs.sh

# Export helm fingerprint before running sensor (the value will be displayed in stdout after running the command above)
export ROX_HELM_CLUSTER_CONFIG_FP="<Helm fingerprint>"

# Make sure you have central's port forward to localhost (e.g. at port 8000)
# Run local-sensor using connected mode
./local-sensor -connect-central "localhost:8000"
```

## How to reproduce the performance tests

### Using the `local-sensor.sh` script

You can run reproducible tests to capture sensor's metrics easily by using the `local-sensor.sh` script. Some recorded metrics can be found [here](https://docs.google.com/spreadsheets/d/1Hq-_9M4fKHy7xljVER01DAMBBQt02N8roh1FE1eK9RA).

1. Define a fake workload ConfigMap called `workload.yaml`:
```yaml
deploymentWorkload:
- deploymentType: Deployment
  lifecycleDuration: 10m0s
  numLabels: 10
  randomLabels: true
  numDeployments: 2500
  numLifecycles: 0
  podWorkload:
    containerWorkload:
      numImages: 0
    lifecycleDuration: 2m0s
    numContainers: 3
    numPods: 5
    processWorkload:
      alertRate: 0.001
      processInterval: 3s
  updateInterval: 5s
networkPolicyWorkload:
- lifecycleDuration: 30m0s
  numNetworkPolicies: 1000
  numLifecycles: 0
  numLabels: 10
  updateInterval: 5s
networkWorkload:
  batchSize: 100
  flowInterval: 1s
nodeWorkload:
  numNodes: 1000
rbacWorkload:
  numBindings: 1000
  numRoles: 1000
  numServiceAccounts: 1000
serviceWorkload:
  numLabels: 10
  numClusterIPs: 300 
  numNodePorts: 300 
  numLoadBalancers: 300
matchLabels: true
numNamespaces: 3
```
2. Build `local-sensor`:
```
./local-sensor.sh --build
```
2. Generate the recorded k8s events file:
```
./local-sensor.sh --generate --with-workload workload.yaml
```
3. Replay the events, and capture metrics:
```
./local-sensor.sh --test
```

These steps will generate five output files located in `tools/local-sensor/out`:

- `trace.jsonl`: Contains the recorded k8s events.
- `time.txt`: Contains the results of the *time* command executed in the test run.
- `local-sensor-cpu-<date>.prof`: Contains the CPU profile of the test run.
- `local-sensor-mem-<date>.prof`: Contains the Memory profile of the test run.
- `sensor_events_dump.json`: Contains information of all the events sent from sensor.
