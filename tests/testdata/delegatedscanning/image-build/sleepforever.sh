#!/bin/bash

# Runs forever and terminates gracefully.

die_func() {
    echo
    echo
    echo "SIG caught, exiting"
    exit 1
}

trap die_func TERM INT

echo "Sleeping forever... (SIGINT or Ctrl-C to exit)"

sleep inf & wait
