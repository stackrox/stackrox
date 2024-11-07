# Bundle Helpers

For hermetic builds with Konflux, we need to provide the full list of resolved dependencies in `requirements.txt`.
These will be prefetched with Cachi2 and made available to the container image build.

## Python Dependency Management

We use Poetry to manage Python dependencies.
Install it from [here](https://python-poetry.org/docs/#installation).
You also need Python==3.6 installed as a prerequisite to keep GHA builds working.
This is going to be solved by [ROX-26860](https://issues.redhat.com/browse/ROX-26860).

In this directory, run

* to add a new dependency: `poetry add PyYAML==6.0`
* to update existing dependencies: `poetry update`

For as long as we build downstream images with CPaaS, you should not add or use dependencies that aren't available OOTB in CPaaS in `render_templates` context.

Check `pyproject.toml` for which dependencies and versions are managed with Poetry.

## Prepare the fully resolved requirements files for Cachi2

1. Generate a fully resolved requirements.txt:

```bash
pip-compile pyproject.toml --generate-hashes > requirements.txt
```

2. Download pip_find_builddeps.py:

```bash
curl -O https://raw.githubusercontent.com/containerbuildsystem/cachito/master/bin/pip_find_builddeps.py
```

3. Generate a fully resolved `requirements-build.txt`:

```bash
./pip_find_builddeps.py requirements.txt \
  --append \
  --only-write-on-update \
  -o requirements-build.in
pip-compile requirements-build.in --allow-unsafe --generate-hashes > requirements-build.txt
```
