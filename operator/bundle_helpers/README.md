# Bundle Helpers

## Dependency Management

Dependencies are managed with Poetry.
This tool can generate the full dependency tree for Python dependencies.
Install it from [here](https://python-poetry.org/docs/#installation).

In this directory, run

* to add a new dependency: `poetry add PyYAML==6.0.2`
* to update existing dependencies: `poetry update`

Check `pyproject.toml` for which dependencies and versions are managed with Poetry.

To regenerate the `requirements.txt` file, run:

```bash
poetry export -o requirements.txt
```
