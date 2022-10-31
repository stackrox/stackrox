# Local-sensor

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
