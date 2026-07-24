---
name: remote-cluster-devloop
description: Build a local StackRox/ACS change into an image, push it to a personal registry, and deploy it to a real remote cluster (e.g. an infractl-provisioned OpenShift cluster) with roxie, then validate the change is actually live. Use when the user asks to "test this on a real cluster", "run my build on the infra cluster", "push my change and deploy it", or "validate this change on OpenShift/a remote cluster". Not for local kind clusters or CI-built images — see "Related devloops" below for those.
disable-model-invocation: true
---

# Remote cluster devloop: build → push → deploy → validate

This skill is for the specific case where **you built the image yourself**
(not CI) and want to run it on a **real remote cluster** you don't control
the node filesystem of — most commonly an OpenShift cluster provisioned with
`infractl`. That combination requires an image registry the cluster can
actually pull from; that's the part this skill exists to make routine and
safe.

## When NOT to use this

- **Local kind cluster**: use `kind load docker-image` instead of a
  registry push — no registry needed at all.
- **CI already built the image**: follow `.claude/skills/test-plan/SKILL.md`
  instead — it deploys a `MAIN_IMAGE_TAG` built by CI (usually from a PR
  bot comment), not a locally-built one.
- **Fast local-only iteration** (no need to prove it on a real cluster yet):
  `make fast-central` / `make fast-sensor` recompile and restart in place,
  much faster than a full image rebuild + push + deploy cycle.
- **roxie usage questions in general**: see `deploy/AGENTS.md` and
  `deploy/README.md` — this skill is one specific use of roxie, not a
  replacement for that documentation.

## Hard rules

1. **Never push an image or deploy to a cluster without first showing the
   exact command (with the real tag/registry/cluster resolved) and getting
   explicit user confirmation.** These are the two remote-state-mutating
   steps in this loop (Step 3 and Step 4 below). Building locally,
   inspecting things, and reading cluster state are not mutating and don't
   need a check-in.
2. Never assume a registry name, cluster kubeconfig path, or namespace —
   confirm them from the user's environment (see Step 0). Don't guess a
   quay.io username.
3. Prefer validation methods that don't require Central admin login when
   possible (e.g. grepping served static assets, checking pod state) —
   faster and fewer secrets to juggle mid-loop.

## Step 0: Establish the environment

Ask the user (or infer from context already established in the
conversation) if not already known:

- Where is the target cluster's kubeconfig? (e.g. downloaded via `infractl
  artifacts <name> -d /tmp/<name>`, giving `/tmp/<name>/kubeconfig`)
- What personal registry should images be pushed to? Default suggestion:
  `quay.io/<their quay.io username>/main` — check `~/.docker/config.json`
  for an existing quay.io login to infer the username, but confirm with the
  user rather than assuming.
- Is that registry repo already public, or does the cluster need an
  image-pull-secret in the `stackrox` namespace? This is a one-time
  decision per cluster/registry, not something to redo every loop
  iteration. If the repo is private and no pull secret exists yet, surface
  this to the user before the first deploy — don't silently fail later on
  `ImagePullBackOff`.
- **What CPU architecture are the cluster's nodes?** Check with
  `oc get nodes -o wide` (see the `KERNEL-VERSION`/image column, or just
  `oc get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}'`).
  Almost all cloud-provisioned clusters (including `infractl` OpenShift) are
  `amd64`. If you're building on an Apple Silicon Mac, `make image` defaults
  to `arm64` and will produce a `Central` pod stuck in `CrashLoopBackOff`
  with `exec ... Exec format error` in its logs — silent otherwise, so check
  this before the first build, not after debugging a crash loop.

## Step 1: Make the change

Whatever code change is being validated. No special handling needed here —
this is a normal edit to the working tree.

## Step 2: Build

```bash
make image GOARCH=amd64   # match the cluster's node arch from Step 0 — omit only if it's genuinely arm64
```

This runs the full `all-builds` (cli, main-build, ui-build) then builds the
`main` image locally, tagged `stackrox/main:$(make tag)` and
`$(DEFAULT_IMAGE_REGISTRY)/main:$(make tag)`. `make tag` is git-describe
based, so every build gets a distinct tag automatically — reuse this same
tag value for the push and deploy steps below rather than inventing one, so
there's never ambiguity about whether the cluster is running stale content.

`GOARCH` defaults to the host machine's architecture (see `make/env.mk`) —
on an Apple Silicon Mac that's `arm64`, which will silently mismatch a
typical `amd64` cloud cluster. `scripts/docker-build.sh` already builds
through `docker buildx --platform linux/$GOARCH`, so cross-compiling for
`amd64` from an arm64 dev machine just works (Go binaries are
`CGO_ENABLED=0`; only the final image assembly runs under buildx/QEMU, not
the compile step), it just isn't the default.

For narrower changes, `AGENTS.md` documents faster options:
`SKIP_UI_BUILD=1` when the change doesn't touch `ui/`, `SKIP_CLI_BUILD=1`
when it doesn't touch `roxctl`. Purely local and safe either way — no
confirmation needed for this step.

## Step 3: Push — ⚠️ mutates remote registry state, confirm first

```bash
TAG=$(make tag)
docker tag stackrox/main:$TAG quay.io/<user>/main:$TAG
docker push quay.io/<user>/main:$TAG
```

Resolve `<user>` and the actual `$TAG` value, show the fully-resolved
commands to the user, and wait for explicit go-ahead before running them.
This is the point where a private/public repo mismatch from Step 0 would
cause a later pull failure — double check it here if it wasn't already
settled.

## Step 4: Deploy — ⚠️ mutates a live cluster, confirm first

Use roxie (the sanctioned deployment tool, see `deploy/AGENTS.md`) with an
image overlay so the `central` Deployment picks up the pushed tag directly,
rather than relying on `MAIN_IMAGE_TAG` (which resolves against the default
registries, not a personal one).

**Use a `--config` file, not `--set`, for the overlay.** roxie's `--set`
flag is parsed as comma-separated `key=value` pairs *before* the value is
YAML-parsed — any comma inside a nested structure (like an overlay's
`patches` list) breaks the parse with a confusing `"key ... has no value"`
error. A config file has no such limit:

```yaml
# /tmp/devloop-roxie-config.yaml
roxie:
  version: <a real, existing tag — see below>

central:
  namespace: stackrox
  spec:
    imagePullSecrets:
      - name: <pull secret name, if using one from Step 0>
    overlays:
      - apiVersion: apps/v1
        kind: Deployment
        name: central
        patches:
          - path: spec.template.spec.containers[name:central].image
            value: quay.io/<user>/main:<TAG>
```

```bash
export KUBECONFIG=<path to the target cluster's kubeconfig>
ROXIE_ENVRC=$(mktemp)
./scripts/roxie.sh deploy central --config /tmp/devloop-roxie-config.yaml --envrc "$ROXIE_ENVRC"
```

Notes:
- `roxie.version` picks the operator/CRD version and must be a tag that
  actually exists (e.g. a recent `X.Y.x-nightly-YYYYMMDD` from `git tag
  --sort=-creatordate`, verified with `docker manifest inspect
  quay.io/rhacs-eng/main:<tag>`) — your locally-built dirty tag won't exist
  there, and that's fine, since the overlay is what actually controls the
  `central` image.
- `scripts/roxie.sh` auto-downloads the pinned version from `ROXIE_VERSION`
  if `roxie` isn't already on `PATH` — no separate install step needed.
- Deploy `central` only when validating something visible from Central
  (including the UI) — no need to also deploy a SecuredCluster for that.
  If the change requires Sensor, deploy `both` (or add a securedcluster
  overlay the same way) instead.
- Confirm the config schema against the installed roxie version if unsure
  — `deploy/AGENTS.md` documents where to check when `roxie --help` or the
  README is out of date for that version.
- Show the fully-resolved config/command (real tag, real kubeconfig path)
  to the user and wait for explicit go-ahead before running it, same as
  Step 3.

## Step 5: Validate

```bash
oc get pods -n stackrox   # or kubectl, same effect
```

Wait for `central` to be `Running`/`Ready`. Then confirm the actual change
landed — don't just trust that the deploy succeeded:

- **UI change**: fetch the served bundle from Central's route and grep for
  a marker string unique to the change. Avoids needing Central admin
  credentials for a quick automated check. A visual check in a browser
  (e.g. via the `browse` skill) is a good second confirmation.
- **Backend/API change**: exercise the relevant endpoint or behavior
  directly.

## Step 6: Iterate

Repeat Steps 1–5 for the next change. Because `make tag` produces a new tag
each time, there's no need to force a rollout restart — a new image
reference is itself sufficient for Kubernetes to roll out the new pod.

## Teardown

```bash
./scripts/roxie.sh teardown
```

Tears down Central/SecuredCluster (not the operator, which roxie manages
automatically). If the cluster itself was provisioned with `infractl`, it
also expires on its own lifespan — teardown of ACS components is about
freeing cluster resources for other testing, not required before the
cluster disappears.

## Related devloops

This is one of several devloop patterns in this repo — pick based on what
you're validating and against what kind of cluster:

| Pattern | Cluster | Image source | When to use |
|---|---|---|---|
| This skill | Real remote cluster (e.g. infractl OpenShift) | Built locally, pushed by you | Validating a local change end-to-end on production-like infra |
| `.claude/skills/test-plan/SKILL.md` | Any cluster with `kubectl`/roxie access | Built by CI (PR bot comment tag) | Manually testing a change that's already gone through CI |
| `make fast-central` / `make fast-sensor` | Existing local dev cluster | N/A (recompile in place) | Fast backend iteration, no fresh image needed |
| `deploy/deploy-local.sh` | Local cluster (e.g. kind) | Built locally, loaded directly (no registry) | Local-only iteration without a real remote cluster |
