# Bundle Helpers

For hermetic builds with Konflux, we need to provide the full list of resolved dependencies in `requirements.txt`.
The dependency source files will be prefetched with Cachi2 and made available to the container image build.
Follow the procedure below after any dependencies change for successful builds in Konflux.

## Prepare the fully resolved requirements files for Cachi2

### Prerequisite

Run the steps inside a container of the same image as the [operator-bundle builder stage](../konflux.bundle.Dockerfile).

```bash
docker run -it -v "$(git rev-parse --show-toplevel)/operator/bundle_helpers:/src" --entrypoint /bin/bash -w /src registry.access.redhat.com/ubi9/python-39:latest
# inside the container
python3 -m pip install pip-tools
```

### Instructions

1. Generate a fully resolved requirements.txt:

```bash
pip-compile requirements.in --generate-hashes
```

2. Download pip_find_builddeps.py:

```bash
curl -fO https://raw.githubusercontent.com/containerbuildsystem/cachito/master/bin/pip_find_builddeps.py
chmod +x pip_find_builddeps.py
```

3. Generate a fully resolved `requirements-build.txt`:

```bash
./pip_find_builddeps.py requirements.txt \
  -o requirements-build.in

pip-compile requirements-build.in --allow-unsafe --generate-hashes
```

4. Exit the container and commit the changes.

For more information, consult the [Cachi2 docs](https://github.com/containerbuildsystem/cachi2/blob/main/docs/pip.md#building-from-source).

### What does each requirements file do?

* `requirements.in`: List of project dependencies.
* `requirements-gha.txt`: The list of project dependencies as required by the build process on GHA and locally. This file exists as a workaround due to a different Python version in this context. Any changes in this or the `requirements.in` file should be synced manually to `requirements-gha.txt`. This file will be deleted after ROX-26860.
* `requirements.txt`: Fully resolved list of all transitive project dependencies.
* `requirements-build.txt`: Fully resolved list of all dependencies required to _build_ the project dependencies from sources in Konflux.
* `requirements-build.in` (not commited): Intermediate result for the generation of `requirements.txt`.
