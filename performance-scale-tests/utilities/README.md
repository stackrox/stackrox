# Prerequisites

Install the following

- kubectl
- infractl
- helm
- roxctl
- Rox Workflow scripts
- jq

Applying load with kubenetbench is optional

# Purpose

Spins up an openshift-4 cluster and repeatedly runs StackRox with different versions of Collector and
reports metrics such as cpu and memory usage for each run.

# Running

The first time you will want to run the following command

```
helm repo add rhacs https://mirror.openshift.com/pub/rhacs/charts
````

To run StackRox on an openshift-4 cluster execute the following


```
./performance-test.sh <json_config_file>
```

The environment variables DOCKER_USERNAME and DOCKER_PASSWORD must be set.

## Config file

The following is an example config file
```
{
        "cluster_name": "jouko-1204",
        "test_dir": "test/plop-open-close-ports-load-stackrox-version/num_ports_0",
        "nrepeat": 5,
        "sleep_after_start_rox": 60,
        "query_window": "10m",
        "num_worker_nodes": 3,
        "load": {
                "load_duration": "600",
                "kubenetbench_load": {
                        "num_streams": 0,
                        "load_test_name": "plop-open-close-ports-load-stackrox-versions"
                }
        },
        "versions": [
                {
                        "nick_name": "plop_enabled",
                        "collector_image_registry": "quay.io/stackrox-io",
                        "collector_image_tag": "3.12.x-32-gd018d81af0",
                        "env_var_file": "env_3.73.x-130-g1ead7d6745.sh",
                        "patch_script": "plop_patch.sh"
                },
                {
                        "nick_name": "plop_disabled",
                        "collector_image_registry": "quay.io/stackrox-io",
                        "collector_image_tag": "3.12.x-32-gd018d81af0",
                        "env_var_file": "env_3.73.x-130-g1ead7d6745.sh",
                        "patch_script": "no_plop_patch.sh"
                }
        ]

}
```
The following is an explanation of the parameters in the config file.

- `cluster_name`: Name of the openshift-4 cluster. If not set it will be set to perf-testing-YY-MM-DD-$RANDOM.


- `test_dir`: The test results will be written to `test_dir` directory. The path of the result files will be
	`test_dir/results_<nick_name>_<run>.txt`. Where `<nick_name>` is specified in the versions section and
	`<run>` is the repeat that the test is on. See below.

- `nrepeat`: How many times are the tests repeated. The default is `5`

- `load-test-name`: If network load is applied this will be the name for the Kubnetbench load test. The Kubenetbench will create a directory with
	results regarding the network load and other files.

- `sleep_after_start_stack_rox`: The time in seconds between when StackRox is started and when the network load is applied. The default is `60`.

- `query_window`: The time window used by the Prometheus queries. The default is `10m`.

- `num_worker_nodes`: The number of work nodes that the cluster is created with

- `teardown_script`: StackRox needs to be torndown between runs and that requires the teardown script
	in the Workflow repo.

- `load`: Parameters concerning the applied load

	- `load_duration`: The duration of the network load in seconds. The default is `600`.
		The time for which the load is applied. After the load stops the deployment is 
		torn down, and stackrox is deployed again with the next set of parameters.
		Even if there is no load this should be set.

	- `kube_burner_load`: Parameters dealing with the kube-burner load
	
		- `config`: The path of the config file used by kube-burner

		- `path`: The path of the kube-burner executable

		- `uuid`: The uuid for the kube-burner runs

	- `kubenetbench_load`: Parameters dealing with the kubenetbench load

		- `num_streams`: The number of streams used by kubenetbench. The higher this is the 
			higher the load. This varies between 0 and 32. If it is set to 0 no
			kubenetbench load will be applied.

		- `load_test_name`: The directory to which the kubenetbench load logs are written to

        - `open_close_ports_load`: Parameters dealing with opening and closing ports. The load is
                applied by repeatedly doing the following. Selecting a random port. If the port is
                closed, start a socat process that listens to it. If the port is open, terminate the
                socat process.

                - `num_ports`: The number of ports in the range of ports which will be opened and closed.
                        E.g. if num_ports is set to 1000, ports in the range 1 to 1000 will be randomly
                        opened and closed.

                - `num_per_second`: The number of ports to be randomly opened and closed per second.
			Please note that this is per pod and that it is also multiplied by num concurrent.

		- `num_pods`: The number of replicas in the load deployment.

		- `num_concurrent`: The number of processes opening and closing ports per pod


- `versions`: This is an array specifying the parameters used for the deployments.

	- `nick_name`: This is the nick name for the set of parameters used for the deployment. It is 
		used in the naming of output files. See test_dir above.

	- `collector_image_registry`: The repository used for the collector image.

	- `collector_image_tag`: The tag of the collector image.

	- `env_var_file`: Contains the environment variables to set the images used for the other 
		components. These environment variables are listed below.

		- Configure central image being used:
		  - `CENTRAL_IMAGE_REGISTRY`
		  - `CENTRAL_IMAGE_NAME`
		  - `CENTRAL_IMAGE_TAG`
		
		- Configure scanner DB image being used:
		  - `SCANNER_DBIMAGE_REGISTRY`
		  - `SCANNER_DBIMAGE_NAME`
		  - `SCANNER_DBIMAGE_TAG`
		
		- Configure scanner image being used:
		  - `IMAGE_MAIN_REGISTRY`
		  - `IMAGE_MAIN_NAME`
		  - `IMAGE_MAIN_TAG`

	- `patch_script`: An additional script that can be run. The purpose is to make patches. It can
		for example be used to change the environment variables in central and collector.


- `artifacts_dir`: Where information about the cluster is stored. This is needed by almost all scripts in
	this directory. If you are running multiple clusters at the same time this needs to be different
	for each cluster. The default is `/tmp/artifacts-<cluster_name>`

- `teardown_script`: The path to the workflow script to teardown stackrox.

# Iterating over the load

There are two scripts to run the performance tests with different load. They are 
loop-through-num-streams.sh, which varies the kubenetbench load and loop-through-num-ports.sh,
which varies the open-close-ports load. The input to both is the same config file as for the main script.

# Output

The test results will be written to `test_dir` directory. The path of the result files will be
`test_dir/results_<nick_name>_<run>.txt`. Where `<nick_name>` is from the third column of the
`<collector_version_file>`. The results are the output of the query.sh script which is called
from the `performance-test.sh` script. 

The output format is in json.


Averaged results over multiple iterations will be saved to `<test_dir>/Average_results_<nick_name>.json`
for each collector version.

# Content

- performance-test.sh: The main entrypoint for the performance tests.

- create-infra.sh: Creates the cluster for the performance tests. 

	Usage: ./create-infra.sh <name> <flavor> <lifespan> <num_worker_nodes>
	
	This is called by performance-test.sh and in that case flavor is hard coded to openshift-4
	and lifespan is hardcoded to 7h

- wait-for-cluster.sh: Waits for the infra cluster to be in the READY state

	Usage: ./wait-for-cluster.sh <cluster_name>

- initialize-kubenetbench.sh: Initializes kubenetbench, which is used for load testing.

	Usage: ./initialize-kubenetbench.sh <artifacts_dir> <load_test_name> <knb_base_dir>

	<load_test_name> is actually the directory where the kubenetbench logs are saved to
	<knb_base_dir> is a directory used by kubenetbench.

- generate-network-load.sh: Generates kubenetbench load

	Usage: ./generate-network-load.sh <artifacts_dir> <load_test_name> <num_streams> <knb_base_dir> <load_duration>

	The greater <num_streams> the greater the load.

- teardown-kubenetbench.sh: Tears down kubenetbench

	Usage: ./teardown-kubenetbench.sh <artifacts_dir> <knb_base_dir>

- start-acs-test-stack.sh: Deploys RHACS using helm

	Usage: ./start-acs-test-stack.sh <cluster_name> <artifacts_dir> <collector_image_registry> <collector_image_tag>

- start-central-and-scanner.sh: Starts central and scanner using helm.

	Usage: ./start-central-and-scanner.sh <artifacts_dir>

- wait-for-pods.sh: Waits for all pods in stackrox namespace to be running.

	Usage: ./wait-for-pods.sh <artifacts_dir>

- get-bundle.sh: Grabs the bundle used to start the secured cluster.

	Usage: ./get-bundle.sh 

- start-secured-cluster.sh: Starts the secured cluster using helm

	Usage: ./start-secured-cluster.sh <artifacts_dir> <collector_image_registry> <collector_image_tag>

- turn-on-monitoring.sh: Turns on prometheus monitoring

	Usage: ./turn-on-monitoring.sh <artifacts_dir>

- query.sh: Performs prometheus queries. The actualy queries are done by query.py. query.sh just 
	gets some things needed to do the queries and passes them to query.py

	Usage: ./query.sh <query_output> <artifacts_dir> <query_window>

	<query_output> is the file to which the query results are written to
	<query_window> is the length of the window of time over which the query is performed. E.g 10m

- get-averages.py: See the section "Getting Averages Over Iterations"


# Getting Averages Over Iterations

To get averaged runs from multiple iterations run

```
	python3 get-averages.py --filePrefix <file_prefix> --numFiles <num_files> --outputFile <output_file>
```

When a new metric is added there is no need to change get-averages.py

### Example

Say the following files were outputted by the query.sh script as a result of running performance-test.sh


TestResults/results_0.json
TestResults/results_1.json
TestResults/results_2.json
TestResults/results_3.json
TestResults/results_4.json
TestResults/results_5.json

To average theses runs something like the following could be executed

```
       python3 get-averages.py --filePrefix TestResults/results_ --numFiles 6 --outputFile TestResults/Average_results.txt
```
