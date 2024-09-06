#!/bin/bash

# This testing script runs forever and terminates gracefully when SIGINT recieved.

die_func() {
    echo
    echo
    echo "SIGINT caught, exiting"
    exit 1
}

trap die_func SIGINT

echo "Sleeping forever... (SIGINT or Ctrl-C to exit)"

sleep inf
