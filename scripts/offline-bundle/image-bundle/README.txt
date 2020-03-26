This bundle contains archives of the images necessary to run the
StackRox Kubernetes Security Platform.

To secure a cluster, your private registry must be able to pull the images.
The import script loads images onto your local Docker engine, but can also
push the images to a private registry.

To install, first reimport the images:
    ./import.sh

See product documentation for complete setup steps. To view documentation:
 - Go to https://help.stackrox.com, or
 - Run import.sh, then follow the instructions displayed.

When you generate your installation configuration and follow the
instructions, use your private registry image names.

See product documentation for additional details.
