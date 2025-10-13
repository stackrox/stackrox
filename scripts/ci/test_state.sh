#!/usr/bin/env bash

# State files used to track test progress across script invocations.

export STATE_IMAGES_AVAILABLE="${SHARED_DIR:-/tmp}/stackrox_ci_state_images_available"
export STATE_BUILD_RESULTS="${SHARED_DIR:-/tmp}/stackrox_ci_state_build_results"
export STATE_DEPLOYED="${SHARED_DIR:-/tmp}/stackrox_ci_state_deployed"

# For the upgrade test
upgrade_progress_state_prefix="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress"
export UPGRADE_PROGRESS_SENSOR_BUNDLE="${upgrade_progress_state_prefix}_sensor_bundle"
export UPGRADE_PROGRESS_UPGRADER="${upgrade_progress_state_prefix}_upgrader"
export UPGRADE_PROGRESS_POSTGRES_PREP="${upgrade_progress_state_prefix}_postgres_prep"
export UPGRADE_PROGRESS_POSTGRES_EARLIER_CENTRAL="${upgrade_progress_state_prefix}_postgres_earlier_central"
export UPGRADE_PROGRESS_POSTGRES_CENTRAL_BOUNCE="${upgrade_progress_state_prefix}_postgres_central_bounce"
export UPGRADE_PROGRESS_POSTGRES_CENTRAL_DB_BOUNCE="${upgrade_progress_state_prefix}_postgres_central_db_bounce"
export UPGRADE_PROGRESS_POSTGRES_MIGRATIONS="${upgrade_progress_state_prefix}_postgres_migrations"
export UPGRADE_PROGRESS_POSTGRES_ROLLBACK="${upgrade_progress_state_prefix}_postgres_rollback"
export UPGRADE_PROGRESS_POSTGRES_SMOKE_TESTS="${upgrade_progress_state_prefix}_postgres_smoke_tests"
