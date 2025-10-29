# Cursor rules

This directory contains a set of [Cursor project rules](https://docs.cursor.com/en/context/rules#project-rules), to steer Cursor's suggestions in ways that benefit the Stackrox development.

## Security

The rules in [security/](security/) provide security guidelines for Cursor to generate secure code/suggestions

- The [security/product-security](security/product-security/) directory is a clone of [Product Security's Cursor rules](https://gitlab.cee.redhat.com/product-security/security-cursor-rules).
- Rules directly in [security/](security/) are Stackrox-specific rules that aren't currently covered by ProdSec rules.
