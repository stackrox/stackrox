# Platform API Tests

This framework is designed to run End-to-End API tests

## How to Run tests

Note: These tests always run on CI unless explicitly labeled otherwise.

### Pre-Test Steps

1. Run `make -C .. proto-generated-srcs` to make sure the protos are up-to-date
1. Deploy the StackRox platform
    ex: `../deploy/k8s/deploy-local.sh`

**Note: There may not be any running deployments called `nginx`, as it will collide with the tests.**

**Note: Some tests are just easier to run on CI (example: anything needing certs).**
Alternatively, try the launcher script in [./e2e](./e2e/).


### From Command Line

1. Add the following values to your ENV  
    export ROX_USERNAME=admin
    export ROX_ADMIN_PASSWORD=$(cat ../deploy/k8s/central-deploy/password)
    export API_ENDPOINT=localhost:8000
1. Run `cd tests; go test -run <TestName>`

### From Goland IDE 

1. From the command line, run `cat ../deploy/k8s/central-deploy/password` and record this.
1. Go to `Run > Edit Configurations`
1. Set Environment Variables to the following:
    `ROX_USERNAME=admin;ROX_ADMIN_PASSWORD=<Value from the first step>;API_ENDPOINT=localhost:8000;<More Env Vars Here>`
1. Save the configuration and Run the test(s)

### Cleanup

1. `unset` any environment variables (or just make a new Terminal window)

## TODO

1. Automated Setup and Cleanup
1. Use a new namespace for any deployments to avoid the nginx clashing.