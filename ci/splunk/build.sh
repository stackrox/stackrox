#! /bin/bash

docker build -t stackrox/splunk-test-repo:6.6.0 .
docker push stackrox/splunk-test-repo:6.6.0
