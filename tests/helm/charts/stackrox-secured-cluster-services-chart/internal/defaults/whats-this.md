`defaults/` directory
======================

This directory provides a set of files that provide a lighter-weight interface for configuring
defaults in the Helm chart, allowing the use of template expressions (including referencing previously
applied defaults) without requiring (an excessive amount of) template control structures (such as
`{{ if kindIs "invalid" ... }}` to determine if a value has already been set).

After applying some "bootstrap" configuration (such as for making available API server resources
visible in a uniform manner), each `.yaml` file in this directory is processed in an order determined
by its name (hence the `NN-` prefixes). Each YAML file consists of multiple documents (separated by
`---` lines) that are rendered as templates and then _merged_ into the effective configuration, giving
strict preference to already set values.

Having a deterministic order is important for being able to rely on previously configured
values (either specified by the user or applied as a default). For example, the file
```yaml
group:
  setting: "foo"
  anotherSetting: 3
---
group:
  derivedSetting: {{ printf "%s-%d" ._rox.group.setting ._rox.group.anotherSetting }}
```
combined with the command-line setting `--set group.setting=bar` will result in the following
"effective" configuration:
```yaml
group:
  setting: "bar"         # user-specified value takes precedence - default value "foo" not applied
  anotherSetting: 3      # default value
  derivedSetting: bar-3  # combination of user-specified value and default value; "pure" default without
                         # any --set arguments would be "foo-3"
```

**Caveats**:
- Templating instructions must be contained to a single document within the multi-document YAML files. In particular,
  the `---` separator must not be within a conditionally rendered block, or emitted by templating code.
- It is recommended to contain dependencies between default settings to a single YAML file. While the `NN-` prefixes
  ensure a well-defined application order of individual files, having dependent blocks in the same file adds clarity.
