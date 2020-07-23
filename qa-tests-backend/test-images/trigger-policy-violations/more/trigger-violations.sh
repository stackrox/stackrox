#!/bin/sh

echo "run something with uid != 0"
su -c ls daemon

sleep 36000
