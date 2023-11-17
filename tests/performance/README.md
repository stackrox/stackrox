# Performance Testing for StackRox - Quick Start

## Go bench tests

In order to run all Go bench tests (including postgres related) use `go-postgres-bench-tests` target.
It could be configured with
- `BENCHTIME` – approximate run time for each benchmark
- `BENCHTIMEOUT` – if positive, sets an aggregate time limit for all tests

```bash
# on base branch
BENCHTIME=1s BENCHTIMEOUT=30m make go-postgres-bench-tests | tee old.txt
# on new branch
BENCHTIME=1s BENCHTIMEOUT=30m make go-postgres-bench-tests | tee new.txt
# compare the results
go install golang.org/x/perf/cmd/benchstat@latest
benchstat -csv old.txt new.txt
```

## Load (aka k6 API tests)

Please use `README.ipynb` for local developnment of performance tests. That README is designed to be easy executable.

### Pre-requirements to use `README.ipynb`

You must install Jupyter `bash` kernel. That can be done with the following commands:
```
pip install bash_kernel
python -m bash_kernel.install
```

After that, you can use VS Code with Jupyter plugin to easy render notebook and execute commands from it.

For that you can install the following plugins:
* https://marketplace.visualstudio.com/items?itemName=ms-toolsai.jupyter
* https://marketplace.visualstudio.com/items?itemName=ms-python.python

Then ensure that the Jupyter server is up and running, open the search bar by hitting `cmd+shift+p` and looking for `Jupyter: Select Interpreter to Start Jupyter Server`.
Once the server is up and running you should be able to configure it with a desired Kernel if not already set. In the top right look for what language is configured then select `Select Another Kernel` -> `Jupyter Kernel` -> `Bash`.

## Scale (aka kube-burner)

See: [scale](scale/README.md)