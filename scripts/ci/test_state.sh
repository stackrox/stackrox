#!/usr/bin/env bash

# State files used to track test progress across script invocations.

export STATE_IMAGES_AVAILABLE="${SHARED_DIR:-/tmp}/stackrox_ci_state_images_available"
export STATE_BUILD_RESULTS="${SHARED_DIR:-/tmp}/stackrox_ci_state_build_results"
export STATE_DEPLOYED="${SHARED_DIR:-/tmp}/stackrox_ci_state_deployed"

# For the upgrade test
export UPGRADE_PROGRESS_SENSOR_BUNDLE="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_sensor_bundle"
export UPGRADE_PROGRESS_UPGRADER="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_upgrader"
export UPGRADE_PROGRESS_LEGACY_PREP="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_legacy_prep"
export UPGRADE_PROGRESS_LEGACY_ROCKSDB_CENTRAL="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_legacy_rocksdb_central"
export UPGRADE_PROGRESS_LEGACY_TO_RELEASE="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_legacy_to_release"
export UPGRADE_PROGRESS_RELEASE_BACK_TO_LEGACY="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_release_back_to_legacy"
export UPGRADE_PROGRESS_POSTGRES_PREP="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_postgres_prep"
export UPGRADE_PROGRESS_POSTGRES_EARLIER_CENTRAL="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_postgres_earlier_central"
export UPGRADE_PROGRESS_POSTGRES_CENTRAL_BOUNCE="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_postgres_central_bounce"
export UPGRADE_PROGRESS_POSTGRES_CENTRAL_DB_BOUNCE="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_postgres_central_db_bounce"
export UPGRADE_PROGRESS_POSTGRES_MIGRATIONS="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_postgres_migrations"
export UPGRADE_PROGRESS_POSTGRES_ROLLBACK="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_postgres_rollback"
export UPGRADE_PROGRESS_POSTGRES_SMOKE_TESTS="${SHARED_DIR:-/tmp}/stackrox_ci_state_upgrade_progress_postgres_smoke_tests"
