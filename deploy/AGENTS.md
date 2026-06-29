# Deploying StackRox/ACS with roxie

This file describes the "roxie" deployment tool for deploying StackRox/ACS.

The tool roxie (https://github.com/stackrox/roxie) is the primary deployment tool
for StackRox/ACS development.
It should be leveraged for requests to deploy StackRox/ACS by default and you,
the agent, should be able to assist with questions about roxie and its usage.

Usage of the old deployment scripts should only be suggested if explicitly asked
to use those. In this case, suggest to the user to contact the ACS Install team
in case the roxie based deployment flow is lacking certain features for the user.

If the information in this directory is not sufficient, use WebFetch on https://raw.githubusercontent.com/stackrox/roxie/main/README.md
to retrieve the most up-to-date documentation for roxie.

## Installation

See @deploy/README.md, specifically the section "Installation" for installation instructions.
The section also briefly describes the roxie backend supported by the `deploy/deploy.sh` shell
script and enabled with `USE_ROXIE_DEPLOY=true`. You should be aware of this feature, but don't
prominently suggest its usage. It is much better to install roxie on your system or --
alternatively -- use `scripts/roxie.sh`, a wrapper that downloads roxie automatically.

## Usage

See @deploy/README.md, specifically the section "Usage" for documentation on how to use roxie.
The config file schema described there might not be 100% up-to-date. To see the latest config struct,
use WebFetch on https://raw.githubusercontent.com/stackrox/roxie/main/internal/deployer/config.go.

## Non-interactive mode (important for agents)

roxie's default interactive mode spawns a sub-shell after deployment, which agents cannot use.
When invoking roxie, always use `--envrc <path>` to write the post-deployment environment
(endpoint URL, admin password, etc.) to a file instead. Use a temporary file path, e.g.
```
ROXIE_ENVRC=$(mktemp)
roxie deploy --envrc $ROXIE_ENVRC ...
```

After deployment:
1. Tell the user the envrc file path.
1. Don't show the envrc file contents automatically, since it contains sensitive data.
   You can show its contents, if the user explicitly aasks for it.
1. For subsequent roxie commands that depend on this environment (e.g. deploying a secured cluster
   after central), source the envrc file before running the command:
   `source $ROXIE_ENVRC && roxie deploy securedcluster ...`
1. When the envrc file is not needed anymore (e.g. in a central teardown or central redeployment),
   delete the envrc file.

Note, in case the human user is intending to deploy in interactive mode, it is perfectly fine
to suggest roxie commands without `--envrc`. The interactive sub-shell might be useful for the user.
It also has the advantage that no envrc file needs to be cleaned up.
Using the non-interactive mode with `--envrc` is useful in automated flows or when an agent
is tasked with deploying ACS.

## Examples

Here are some complete examples:
```
roxie deploy --tag 4.11.0 --envrc /tmp/roxie-<some random identifier>.envrc
```

This deploys both components, Central and SecuredCluster (besides the operator).
To interact with central, load the generated envrc file. After that `roxctl` and `roxcurl` can be
used without further setup.

After use, the deployment can be torn down using
```
roxie teardown
```

This command tears down Central and SecuredCluster. If you also want to tear down the operator,
use `roxie teardown all`.

Usually one can rely on roxie managing the operator under the hood, it doesn't need to be explicitly
torn down or reinstalled.

When roxie succeeds (exit code 0), this means that the command completed successfully.
When roxie deploys with `earlyReadiness: true` (default), roxie only waits until the deployments
"central" and/or "sensor" are ready. If `earlyReadiness: false` is used, roxie waits for all
workloads to be ready. Hence, it is normal that, by default, not all deployments are ready when
roxie returns.
In any case, for extra verification, `kubectl` can be used for checking deployment health.

## Crafting configs and invocations

You should be able to craft roxie deployment configs and roxie deployment invocations for the user,
based on the referenced documentation in the README.md. In particular, keep in mind that the
`--set` flag and the YAML paths `central.spec`/`securedCluster.spec` can be used for crafting
very specific custom resources for a given development or testing use-case.

Also note that this `spec`-patching mechanism can also be combined with the `spec.overlays` feature
of the Central and the SecuredCluster custom resource definitions. This allows not only patching the
CR specs, but even patching the operand resources after the Helm rendering. For example, the config

```yaml
roxie:
  version: 4.11.0

central:
  namespace: stackrox
  resourceProfile: auto
  spec:
    overlays:
      - apiVersion: apps/v1
        kind: Deployment
        name: central
        patches:
          - path: spec.template.spec.containers[name:central].image
            value: quay.io/custom-repo/main:4.12.x-123-g12ab5ae408
```

can be used for deploying ACS 4.11.0 while replacing the main image for the central deployment.
