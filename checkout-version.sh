#!/usr/bin/env bash
set -eo pipefail

git checkout 3.70.0 -- image/templates/helm/
cp -r image/templates/helm/ image/templates/helm-3.70.0/

git checkout 3.69.0 -- image/templates/helm/
cp -r image/templates/helm/ image/templates/helm-3.69.0/

git checkout HEAD -- image/templates/helm
