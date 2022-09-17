# Roxctl e2e Bats tests

## Prerequisites

In order to run the tests locally, we need to install Bats (in version at least 1.5.0) and two helpers: `bats-support` and `bats-assert`.

Bats can be installed with:

```shell
$ brew install bats-core
# (output suppressed)

$ bats --version
Bats 1.5.0
```

[other installation methods](https://bats-core.readthedocs.io/en/stable/installation.html) are also available.

The helpers should be installed from sources and placed into `$HOME/bats-core/`.
Installation:

```shell
mkdir -p "$HOME/bats-core/"
git -C "$HOME/bats-core/" clone --depth=1 https://github.com/bats-core/bats-core
git -C "$HOME/bats-core/" clone --depth=1 https://github.com/bats-core/bats-assert
git -C "$HOME/bats-core/" clone --depth=1 https://github.com/bats-core/bats-support
(cd "$HOME/bats-core/bats-core" && sudo ./install.sh "/usr/local")
bats --version
```

The helpers are installed correctly if the following tests are passing:

```shell
test -f "$HOME/bats-core/bats-support/load.bash"
test -f "$HOME/bats-core/bats-assert/load.bash"
```

alternatively, one can create a trivial bats test-case and try to run it:

```bash
$ cat case.bats
#!/usr/bin/env bats

load "$HOME/bats-core/bats-support/load.bash"
load "$HOME/bats-core/bats-assert/load.bash"

@test "installation check" {
  run echo "quick fox jumped over the lazy dog"
  assert_success
  assert_output --partial "fox jumped over"
  refute_output --partial "lorem ipsum dolor"
}

$ chmod a+x case.bats
$ ./case.bats
 ✓ installation check

1 test, 0 failures
```

## Running locally

To run the tests locally, you may simply execute a bats file:

```shell
./tests/roxctl/bats-tests/roxctl-central-development.bats
```

or run the entire suite:

```shell
bats --recursive tests/roxctl/bats-tests/
```
