---
name: remote-cluster-devloop
description: Build a local StackRox/ACS change into an image, push it to a personal registry, and deploy it to a remote cluster. Use when the user asks to "test this on a real cluster", "deploy my change", or "validate on OpenShift". Not for local kind clusters or CI-built images.
disable-model-invocation: true
---

# Remote cluster devloop: build → push → deploy → validate

Build a local change, push it to a personal registry, and deploy it to a remote cluster (typically an infractl-provisioned OpenShift cluster). Iterate until the change is validated.

## When NOT to use this

- **Local kind cluster**: use `kind load docker-image` — no registry needed.
- **CI already built the image**: deploy with `MAIN_IMAGE_TAG` from the CI bot comment.
- **Fast local iteration**: `make fast-central` / `make fast-sensor` recompile in place without a full image rebuild.

## Hard rules

1. **Never push an image or deploy to a cluster without showing the exact command and getting explicit user confirmation.** Building locally and reading cluster state are fine without check-in.
2. Never assume a registry name, kubeconfig path, or namespace — confirm from the user's environment.

## Step 0: Establish the environment

Confirm (or infer from conversation context):

- **Kubeconfig**: e.g. `/tmp/<cluster-name>/kubeconfig` from `infractl artifacts`.
- **Personal registry**: e.g. `quay.io/<username>/main`. Check `~/.docker/config.json` for an existing quay.io login to suggest a username, but confirm.
- **Registry visibility**: is the repo public, or does the cluster need an image-pull-secret? Settle this before the first deploy.

## Step 1: Make the change

Normal code edit. No special handling.

## Step 2: Build

```bash
make image
```

This builds all binaries and assembles the `main` image, tagged `stackrox/main:$(make tag)`. The tag is git-describe based, so every build gets a distinct tag automatically. Use this same tag value for push and deploy.

For narrower changes: `SKIP_UI_BUILD=1` when not touching `ui/`, `SKIP_CLI_BUILD=1` when not touching `roxctl`.

No confirmation needed — purely local.

## Step 3: Push — confirm first

```bash
TAG=$(make tag)
docker tag stackrox/main:$TAG quay.io/<user>/main:$TAG
docker push quay.io/<user>/main:$TAG
```

Show the fully-resolved commands and wait for go-ahead.

## Step 4: Deploy — confirm first

Use roxie to deploy with the custom image. See `deploy/AGENTS.md` for roxie usage details.

The key requirement is overriding the Central (and/or Sensor) image to point at the personal registry tag from Step 3. The exact mechanism depends on roxie version — check `deploy/AGENTS.md` and `roxie --help` for the current overlay/config syntax.

```bash
export KUBECONFIG=<path>
./scripts/roxie.sh deploy central [with image override to quay.io/<user>/main:<TAG>]
```

Deploy `central` only when validating Central/UI changes. If the change requires Sensor, deploy `both` or add a securedcluster config.

Show the fully-resolved command to the user and wait for go-ahead.

## Step 5: Validate

```bash
oc get pods -n stackrox
```

Wait for pods to be `Running`/`Ready`. Then confirm the change actually landed:

- **UI change**: fetch served assets from Central's route and grep for a marker string, or check in a browser.
- **Backend/API change**: exercise the relevant endpoint or behavior directly.
- **Feature-flagged change**: ensure the feature flag env var is set on the deployment (e.g. `ROX_POLICY_REPORTS=true`).

## Step 6: Iterate

Repeat Steps 1–5. `make tag` produces a new tag each time, so Kubernetes will roll out the new pod automatically — no force-restart needed.

## Teardown

```bash
./scripts/roxie.sh teardown
```

Tears down ACS components. The infractl cluster itself expires on its own lifespan.
