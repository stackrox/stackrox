#!/usr/bin/env python3
"""Pin ClairCore to the version selected by its current main branch."""

import subprocess
import sys

MODULE = "github.com/quay/claircore"


def _run(command, runner, check=True):
    result = runner(
        command,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if check and result.returncode:
        if result.stdout:
            print(result.stdout, file=sys.stderr, end="")
        if result.stderr:
            print(result.stderr, file=sys.stderr, end="")
        raise subprocess.CalledProcessError(
            result.returncode,
            command,
            output=result.stdout,
            stderr=result.stderr,
        )
    return result


def _module_version(ref, runner):
    module = f"{MODULE}@{ref}" if ref else MODULE
    result = _run(["go", "list", "-m", "-f={{.Version}}", module], runner)
    version = result.stdout.strip()
    if not version:
        raise RuntimeError(f"Go returned no version for {module}")
    return version


def update_claircore(runner=subprocess.run):
    """Update go.mod/go.sum only when ClairCore main has a different revision."""
    upstream_version = _module_version("main", runner)
    current_version = _module_version(None, runner)
    if upstream_version == current_version:
        return False

    go_get = _run(["go", "get", f"{MODULE}@main"], runner)
    if go_get.stdout:
        print(go_get.stdout, file=sys.stderr, end="")
    if go_get.stderr:
        print(go_get.stderr, file=sys.stderr, end="")

    diff = _run(["git", "diff", "--quiet", "--", "go.mod", "go.sum"], runner, check=False)
    if diff.returncode == 0:
        return False
    if diff.returncode == 1:
        return True
    if diff.stdout:
        print(diff.stdout, file=sys.stderr, end="")
    if diff.stderr:
        print(diff.stderr, file=sys.stderr, end="")
    raise subprocess.CalledProcessError(
        diff.returncode, diff.args, diff.stdout, diff.stderr
    )


def main():
    print("true" if update_claircore() else "false")


if __name__ == "__main__":
    main()
