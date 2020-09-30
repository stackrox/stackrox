# StackRox Kubernetes Security Platform - Central Services Helm Chart

This Helm chart allows you to deploy the central services of the StackRox
Kubernetes Security Platform: StackRox Central and StackRox Scanner.

## Prerequisites

To deploy the central services for the StackRox Kubernetes Security platform
using Helm, you must:
- Have at least version 3.1 of the Helm tool installed on your machine
- Have credentials for the `stackrox.io` registry or the other image registry
  you use.

## Add the Canonical Chart Location as a Helm Repository

The canonical repository for StackRox Helm charts is https://charts.stackrox.io.
To use StackRox Helm charts on your machine, run
```sh
helm repo add stackrox https://charts.stackrox.io
```
This command only needs to be run once on your machine. Whenever you are deploying
or upgrading a chart from a remote repository, it is advisable to run
```sh
helm repo update
```
beforehand.

## Deploy Central Services Using Helm

The basic command for deploying the central services is
```sh
helm install -n stackrox --create-namespace \
    stackrox-central-services stackrox/central-services
```
If you have a copy of this chart on your machine, you can also reference the
path to this copy instead of `stackrox/central-services` above.

In order to be able to access StackRox Docker images, you also need image pull
credentials. There are several ways to inject the required credentials (if any)
into the installation process:
- **Explicitly specify username and password:** Use this if you are using the images
  from the default registry (`stackrox.io`), or a registry that supports username/password
  authentication. Pass the following arguments to the `helm install` command:
  ```sh
  --set imagePullSecrets.username=<registry username> --set imagePullSecrets.password=<registry password>
  ```
- **Use pre-existing image pull secrets:** If you already have one or several image pull secrets
  created in the namespace to which you are deploying, you can reference these in the following
  way (we assume that your secrets are called `pull-secret-1` and `pull-secret-2`):
  ```sh
  --set imagePullSecrets.useExisting="pull-secret-1;pull-secret-2"
  ```
- **Do not use image pull secrets:** If you are pulling your images from a registry in a private
  network that does not require authentication, or if the default service account in the namespace
  to which you are deploying is already configured with appropriate image pull secrets, you do
  not need to specify any additional image pull secrets. To inform the installer that it does
  not need to check for specified image pull secrets, pass the following option:
  ```sh
  --set imagePullSecrets.allowNone=true
  ```
  
### Accessing the StackRox Portal After Deployment

Once you have deployed the StackRox Kubernetes Security Platform Central Services via
`helm install`, you will see an information text on the console that contains any things to
note, or warnings encountered during the installation text. In particular, it instructs you
how to connect to your Central deployment via port-forward (if you have not configured an
exposure method, see below), and the administrator password to use for the initial login.

### Applying Custom Configuration Options

This Helm chart has many different configuration options. For simple use cases, these can be
set directly on the `helm install` command line; however, we generally recommend that you
store your configuration in a dedicated file.

#### Using the `--set` family of command-line flags

This approach is the quickest way to customize the deployment, but it does not work for
more complex configuration settings. Via the `--set` and `--set-file` flags, which need to be
appended to your `helm install` invocation, you can inject configuration values into the
installation process. Here are some examples:
- **Specify the StackRox license key upon deployment:** This eliminates the need to upload a
  license file via the StackRox portal upon the initial login.
  ```sh
  --set-file licenseKey=path/to/license.lic
  ```
- **Deploy StackRox in offline mode:** This configures StackRox in a way such that it will not
  reach out to any external endpoints.
  ```sh
  --set env.offlineMode=true
  ```
- **Configure a fixed administrator password:** This sets the password with which you log in to
  the StackRox portal as an administrator. If you do not configure a password yourself, one will
  be created for you and printed as part of the installation notes.
  ```sh
  --set central.adminPassword.value=mysupersecretpassword
  ```

#### Using configuration YAML files and the `-f` command-line flag

To ensure the best possible upgrade experience, it is recommended that you store all custom
configuration options in two files: `values-public.yaml` and `values-private.yaml`. The former
contains all non-sensitive configuration options (such as whether to run in offline mode), and the
latter contains all sensitive configuration options (such as the administrator password, or
custom TLS certificates). The `values-public.yaml` file can be stored in, for example, your Git
repository, while the `values-private.yaml` file should be stored in a secrets management
system.

There is a large number of configuration options that cannot all be discussed in minute detail
in this README file. However, the Helm chart contains example configuration files
`values-public.yaml.example` and `values-private.yaml.example`, that list all the available
configuration options, along with documentation. The following is just a brief example of what
can be configured via those files:
- **`values-public.yaml`:**
  ```yaml
  env:
    offlineMode: true  # run in offline mode
  
  central:
    # Use custom resource overrides for central
    resources:
      requests:
        cpu: 4
        memory: "8Gi"
      limits:
        cpu: 8
        memory: "16Gi"
  
    # Expose central via a LoadBalancer service
    exposure:
      loadBalancer:
        enabled: true
  
  scanner:
    # Run without StackRox Scanner (NOT RECOMMENDED)
    disable: true
  
  customize:
    # Apply the important-service=true label for all objects managed by this chart.
    labels:
      important-service: true
    # Set the CLUSTER=important-cluster environment variable for all containers in the
    # central deployment:
    central:
      envVars:
        CLUSTER: important-cluster
  ```
- **`values-private.yaml`**:
  ```yaml
  licenseKey: |
    ... the StackRox license key ...
  
  central:
    # Configure a default TLS certificate (public cert + private key) for central
    defaultTLS:
      cert: |
        -----BEGIN CERTIFICATE-----
        MII...
        -----END CERTIFICATE-----
      key: |
        -----BEGIN EC PRIVATE KEY-----
        MHc...
        -----END EC PRIVATE KEY-----
  ```

After you have created these YAML files, you can inject the configuration options into the
installation process via the `-f` flag, i.e., by appending the following options to the
`helm install` invocation:
```sh
-f values-public.yaml -f values-private.yaml
```

### Changing Configuration Options After Deployment

If you wish to make any changes to the deployment, simply change the configuration options
in your `values-public.yaml` and/or `values-private.yaml` file(s), and inject them into an
`helm upgrade` invocation:
```sh
helm upgrade -n stackrox stackrox-central-services stackrox/central-services \
    -f values-public.yaml \
    -f values-private.yaml
```
Under most circumstances, you will not need to supply the `values-private.yaml` file, unless
you want changes to sensitive configuration options to be applied.

Of course you can also specify configuration values via the `--set` or `--set-file` command-line
flags. However, these options will be forgotten with the next `helm upgrade` invocation, unless
you supply them again.
