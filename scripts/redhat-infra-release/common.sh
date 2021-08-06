#!/usr/bin/env bash

info() {
    echo >&2 "$@"
}

die() {
    echo >&2 "$@"
    exit 1
}
