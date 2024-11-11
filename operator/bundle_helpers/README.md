# Bundle Helpers

For hermetic builds with Konflux, we need to provide the full list of resolved dependencies in `requirements.txt`.
The dependency source files will be prefetched with Cachi2 and made available to the container image build.
Because GHA/upstream is running on a different Python version, it uses `requirements-upstream.txt` to mange its own dependencies.
Follow the procedure below after any dependencies change for successful builds in Konflux and mirror those changes into the `requirements-upstream.txt`.

## Prepare the fully resolved requirements files for Cachi2

### Prerequisite

Install [pip-compile](https://pip-tools.readthedocs.io/en/stable/).

Example:

```bash
python3 -m pip install --user pipx
python3 -m pipx ensurepath
pipx install pip-tools
```

### Instructions

1. Generate a fully resolved requirements.txt:

```bash
pip-compile requirements.in --generate-hashes > requirements.txt
```

2. Download pip_find_builddeps.py:

```bash
curl -fO https://raw.githubusercontent.com/containerbuildsystem/cachito/master/bin/pip_find_builddeps.py
chmod +x pip_find_builddeps.py
```

3. Generate a fully resolved `requirements-build.txt`:

```bash
./pip_find_builddeps.py requirements.txt \
  --append \
  --only-write-on-update \
  -o requirements-build.in

pip-compile requirements-build.in --allow-unsafe --generate-hashes > requirements-build.txt
```

For more information, consult the [Cachi2 docs](https://github.com/containerbuildsystem/cachi2/blob/main/docs/pip.md#building-from-source).

### What does each requirements file do?

* `requirements.in`: List of project dependencies.
* `requirements-gha.txt`: Temporary list of project dependencies as required by the build process on GHA. This will be deleted after ROX-26860.
* `requirements.txt`: Fully resolved list of all transitive project dependencies.
* `requirements-build.txt`: Fully resolved list of all dependencies required to _build_ the project dependencies from sources in Konflux.
* `requirements-build.in` (not commited): Intermediate result for the generation of `requirements.txt`.
