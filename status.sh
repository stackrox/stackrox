#!/bin/bash

echo STABLE_MAIN_VERSION $(git describe --tags --abbrev=10 --dirty)
echo STABLE_COLLECTOR_VERSION $(cat COLLECTOR_VERSION)
echo STABLE_CLAIRIFY_VERSION $(cat CLAIRIFY_VERSION)
