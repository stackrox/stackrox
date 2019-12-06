#! /bin/bash

docker build -t stackrox/splunk-test-repo:latest .
docker push stackrox/splunk-test-repo:latest
