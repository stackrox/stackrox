#!/usr/bin/env bash

# curl -X POST http://example.com/form -H "Content-Type: application/json" -d '{"username":"example_user", "password":"example_pass"}'

roxcurl /v1/notifiers -X POST -d '{"name": "sentinel", "type": "microsoft_sentinel", "ui_endpoint": "asdf"}'
