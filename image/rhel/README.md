# RedHat Based main image

The RedHat based main image is currently used for the RedHat marketplace as well as for DoD customers.

This image is built in opinionated way based on the DoD Centralized Artifacts Repository (DCAR) requirements outlined [here](https://dccscr.dsop.io/dsop/dccscr/tree/master/contributor-onboarding)

## Adding new files to the main-rhel container

To add a new file artifact to the rhel container, include it in `create-bundle.sh` script, do not add it to the Dockerfile in this directory.
