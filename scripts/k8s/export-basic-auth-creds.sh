#!/bin/sh

# Finds basic auth credentials in the deploy dir and exports ROX_USERNAME and 
# ROX_PASSWORD. The script is intended to be sourced (for the parent shell to
# get access to the exported vars). Deploy dir is passed as a parameter.

usage() {
  echo "Usage: $0 <deploy_dir>"
  exit 2
}

DEPLOY_DIR=$1
[ -n "${DEPLOY_DIR}" ] || usage

if [ -f "${DEPLOY_DIR}/central-deploy/password" ]; then
    password="$(cat "${DEPLOY_DIR}"/central-deploy/password)"
    export ROX_USERNAME=admin
    export ROX_PASSWORD="$password"
else
    echo "Expected to find file ${DEPLOY_DIR}/central-deploy/password"
    exit 1
fi
