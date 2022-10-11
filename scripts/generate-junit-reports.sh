#!/usr/bin/env bash

mkdir -p junit-reports

go-junit-report <"test-output/test.log" >"junit-reports/report.xml"
env
cat /logs/process-log.txt
