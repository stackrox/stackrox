# Bundle Helpers

## Dependency Management

Dependencies are managed with Poetry.
This tool can generate the full dependency tree for Python dependencies.
Install it from [here](https://python-poetry.org/docs/#installation).
You also need Python==3.6 installed as a prerequisite to keep GHA builds working.
This is going to be solved by [ROX-26860](https://issues.redhat.com/browse/ROX-26860).

In this directory, run

* to add a new dependency: `poetry add PyYAML==6.0`
* to update existing dependencies: `poetry update`

Check `pyproject.toml` for which dependencies and versions are managed with Poetry.

To regenerate the `requirements.txt` file, run:

```bash
poetry export -o requirements.txt
```
