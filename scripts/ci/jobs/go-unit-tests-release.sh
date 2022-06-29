#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GOTAGS="release" "${ROOT}/go-unit-tests.sh"
