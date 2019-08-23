This bundle contains archives of the images necessary to run the StackRox
Container Security Platform.

To install, first reimport the images:
    ./import.sh

Then, generate your installation configuration and follow the
instructions presented, using your private registry image names:
    roxctl central generate interactive > stackrox.zip

See product documentation for additional details.
