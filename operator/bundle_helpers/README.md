# Bundle Helpers

## Python Dependency Management

For hermetic builds with Konflux, we need to provide the full list of resolved dependencies in `requirements.txt`.
These will be prefetched with Cachi2 and made available to the container image build.

We use Poetry to manage Python dependencies and for generating the complete `requirements.txt`.
This tool can generate the full dependency tree for Python dependencies.
Install it from [here](https://python-poetry.org/docs/#installation).
You also need Python==3.6 installed as a prerequisite to keep GHA builds working.
This is going to be solved by [ROX-26860](https://issues.redhat.com/browse/ROX-26860).

In this directory, run

* to add a new dependency: `poetry add PyYAML==6.0`
* to update existing dependencies: `poetry update`

For as long as we build downstream images with CPaaS, you should not add or use dependencies that aren't available OOTB in CPaaS in `render_templates` context.

Check `pyproject.toml` for which dependencies and versions are managed with Poetry.

To regenerate the `requirements.txt` file, run:

```bash
poetry export -o requirements.txt
```
