Package `helmtest`
======

The `helmtest` package allows you to declaratively specify test suites for Helm charts. It specifically
seeks to address two inconveniences of the "normal" Go unit test-based approach:
- it allows testing a multitude of different configurations via a hierarchical, YAML-based specification
  of test cases.
- it makes writing assertions about the generated Kubernets objects easy, by using `jq` filters as the
  assertion language.
  
Format of a test
=========
A `helmtest` test is generally defined in a YAML file according to the format specified in `spec.go`.
Tests are organized in a hierarchical fashion, in the sense that a test may contain one or more
sub-tests. Tests with no sub-tests are called "leaf tests", and other tests are called "non-leaf tests".
A Helm chart is only rendered and checked against expectations in leaf tests; in such a setting,
the leaf test inherits certain properties from its non-leaf ancestors.

The general schema of a test is as follows:
```yaml
name: "string" # the name of the test (optional but strongly recommended). Auto-generated if left empty.
release:  # Overrides for the Helm release properties. These are applied in root-to-leaf order.
  name: "string"  # override for the Helm release name
  namespace: "string"  # override for the Helm release namespace
  revision: int # override for the Helm release revision
  isInstall: bool # override for the "IsInstall" property of the release options
  isUpgrade: bool # override for the "IsUpgrade" property of the release options
server:
  visibleSchema: # openAPI schema which is visible to helm, i.e. to check API resource availability
  # all valid schemas are:
  - kubernetes-1.20.2
  - openshift-3.11.0
  - openshift-4.1.0
  - com.coreos
  availableSchemas: [] # openAPI schema to validate against, i.e. to validate if rendered objects could be applied
values:  # values as consumed by Helm via the `-f` CLI flag.
  key: value
set:  # alternative format for Helm values, as consumed via the `--set` CLI flag.
  nes.ted.key: value
defs: |
  # Sequence of jq "def" statements. Each def statement must be terminated with a semicolon (;). Defined functions
  # are only visible in this and descendant scopes, but not in ancestor scopes.
  def name: .metadata.name;

expectError: bool # indicates whether we can tolerate an error. If unset, inherit from the parent test, or
                  # default to `false` at the root level.
expect: |
  # Sequence of jq filters, one per line (or spanning multiple lines, where each continuation line must begin with a
  # space).
  # See the below section on the world model and special functions.
  .objects[] | select(.metadata?.name? // "" == "")
    | assertNotExist  # continuation line
tests: []  # a list of sub-tests. Determines whether the test is a leaf test or non-leaf test.
```

A comprehensive set of hierarchically organized tests to be run against a Helm chart is called a "suite". Each suite
is defined in a set of YAML files located in a single directory on the filesystem (a directory may hold at most one
suite). The properties of the top-level test in the suite (such as a common set of expectations or Helm values to be
inherited by all tests) can be specified in a `suite.yaml` file within this directory. The `suite.yaml` file may be
absent, in which case there are no values, definitions, expectations etc. shared among all the tests in the suite. In
addition to the tests specified in the `tests:` stanza of the `suite.yaml` file (if any), the tests of the suite are
additionally read from files with the extension `.test.yaml` in the suite directory. Note that any combination of
defining tests in the `suite.yaml` and in individual files may be used, these tests will then be combined. In
particular, it is possible to define arbitrary test suites either with only `.test.yaml` files, with only a `suite.yaml`
file, or with combinations thereof.

Inheritance
----------------
For most fields in the test specification, a test will inherit the value from its parent test (which might use an
inherited value as well, etc.). If an explicit value is given, this value
- overrides the values from the parent for the following fields: `expectError` and the individual sub-fields of
  `release`.
- is merged with the values from the parent for the following fields: `values`, `set` (in such a way that the values
  from the child take precedence).
- is added to the values from the parents for the following fields: `expect`, `defs`.

World model
============

As stated above, expectations are written as `jq` filters (using `gojq` as an evaluation engine). Generally, a filter
that evaluates to a "falsy" value is treated as a violation. In contrast to normal JS/`jq` semantics, an empty list,
object, or string will also be treated as "falsy". The input to those filters is a JSON object with the following
fields:
- `helm` contains the render values passed to Helm, such as `.Values`, `.Release`, `.Capabilities`, `.Chart` etc.
- `error` contains the rendering error message, if any. This will only be set when an error occurred. Note that if you
  want to check assertions on the error message, you _must_ still set `expectError: true`, otherwise the test will fail.
- `objects` contains all Kubernetes objects from all rendered YAML files as a JSON array. This will only be set when
  no error occurred.
- `notes` contains the output of the rendered `NOTES.txt`. This will only be set when no error occurred.

In addition, for every Kubernetes object kind, there will be an entry using the lowercase plural form of the object
kind as the key, and containing a name-indexed object of all Kubernetes objects as the value. To locate the deployment
"sensor", you can thus either write
`.objects[] | select(.metadata.kind == "Deployment" and .metadata.name == "sensor")`, or simply `.deployments.sensor`.

Special functions
===============

In addition to the standard `jq` functions, you can use the following ones:
- `fromyaml`, `toyaml` - the equivalent of `fromjson` and `tojson` for YAML.
- `assertNotExist` - this function will fail if ever executed. Semantically equivalent to just writing `false`, but
  with the advantage that the offending object is printed.
- `assertThat(f)` - asserts that a filter `f` holds for the input object. If `. | f` evaluates to `false`, this will
  print the value of `.` as well as the original string representation of `f`. Hence, while `... | .name == "foo"` and
  `| assertThat(.name == "foo")` are semantically equivalent, the latter is preferable as it is much easier to debug.
- `assumeThat(f)` - assumes that a filter `f` holds for the input object. If it doesn't, the evaluation is aborted for
  the given input object, and no failure is triggered.
- `print` - prints input directly with `fmt.Println` and returns it, i.e. to print all objects in a test as
   `yaml` write `.objects[] | toyaml | print`.
