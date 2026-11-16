# Platform API Tests

This framework is designed to run End-to-End API tests

## How to Run tests

Note: These tests always run on CI unless explicitly labeled otherwise.

### Pre-Test Steps

1. Run `make -C .. proto-generated-srcs` to make sure the protos are up-to-date
1. Deploy the StackRox platform
    ex: `../deploy/deploy-local.sh`

**Note: There may not be any running deployments called `nginx`, as it will collide with the tests.**

**Note: Some tests are just easier to run on CI (example: anything needing certs).**
Alternatively, try the launcher script in [./e2e](./e2e/).


### From Command Line

1. Deploy using roxie (`./deploy/deploy.sh`), which sets up the roxie environment automatically
1. Within the roxie subshell, add the following values to your ENV  
    export ROX_USERNAME=admin
    # ROX_ADMIN_PASSWORD is already set by roxie
    export API_ENDPOINT=localhost:8000
1. Run `cd tests; go test -run <TestName>`

Alternatively, if using `--envrc` mode: `source <envrc-file>` first to load the credentials.

### From Goland IDE 

1. Deploy using roxie with `--envrc`: `./deploy/deploy.sh --envrc /tmp/roxie.env`
1. From the command line, run `source /tmp/roxie.env && echo $ROX_ADMIN_PASSWORD` and record this.
1. Go to `Run > Edit Configurations`
1. Set Environment Variables to the following:
    `ROX_USERNAME=admin;ROX_ADMIN_PASSWORD=<Value from the first step>;API_ENDPOINT=localhost:8000;<More Env Vars Here>`
1. Save the configuration and Run the test(s)

### Cleanup

1. `unset` any environment variables (or just make a new Terminal window)

## TODO

1. Automated Setup and Cleanup
1. Use a new namespace for any deployments to avoid the nginx clashing.