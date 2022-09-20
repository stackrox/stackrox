# GitHub Actions

## Upstream Release Automation

### General Rules

* **Ensure reentrancy**: expect a workflow to be re-run from failed state;
* **Suggest the next step** if it is not automated: print `:arrow_right:` emoji
  with the suggested action to `$GITHUB_STEP_SUMMARY` or post a message to Slack;
* **Highlight major things**: print to stdout with `::error::`, `::warning::` or
  `::notice::` prefix to higlight important status. Markdown is not supported;
* **Log minor things**: print to `$GITHUB_STEP_SUMMARY` with markdown to describe
  the executed actions and suggest the next step;
* **Support dry-run**: use the `DRY_RUN` environment variable to check for dry-run,
  it holds `true` or `false` values;
* **Extract large scripts**: look in the `scripts` folder for examples;
