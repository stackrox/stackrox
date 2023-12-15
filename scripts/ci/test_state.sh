#!/usr/bin/env bash

# State files used to track test progress across script invocations.

export STATE_IMAGES_AVAILABLE="${SHARED_DIR:-/tmp}/stackrox_ci_state_images_available"
export STATE_DEPLOYED="${SHARED_DIR:-/tmp}/stackrox_ci_state_deployed"
