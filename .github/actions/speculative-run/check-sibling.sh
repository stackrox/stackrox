#!/bin/bash
# Check if sibling matrix job completed — called between expensive steps.
# If sibling is done, skip remaining work.
if [[ -f /dev/shm/sibling-completed ]]; then
  echo "::notice::Sibling job completed first. Skipping remaining steps."
  echo "skip=true" >> "$GITHUB_OUTPUT"
else
  echo "skip=false" >> "$GITHUB_OUTPUT"
fi
