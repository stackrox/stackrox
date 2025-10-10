# Feature Flag Removal Plan: ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT

## Removal Sequence

**📍 REMOVAL ORDER: #1 of 4 (FIRST - Foundation)**

This flag should be removed **first** in the sequence because:
1. ✅ **Foundation layer:** Enables connection retry mechanism that other features depend on
2. ✅ **Simplest change:** Cleanest code paths, lowest complexity
3. ✅ **Builds confidence:** Success here validates the removal process
4. ✅ **No dependencies:** Does not depend on other removals

**What depends on this:**
- ROX_CAPTURE_INTERMEDIATE_EVENTS (requires sensor to stay alive for buffering)
- ROX_SENSOR_RECONCILIATION (requires sensor to stay alive for reconciliation)

**PR Strategy:** Create as standalone PR on `master` branch.

## Overview

**Feature Flag:** `ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT`  
**Enabled Since:** Release 4.1 (June 2023) - **~2.3 years**  
**Last Change:** Release 4.1 (single clean toggle: disabled → enabled)  
**Risk Assessment:** Medium  
**Estimated Effort:** 4-6 hours

## Executive Summary

This feature flag enables a new behavior in Sensor where it avoids restarting when the gRPC connection with Central ends, instead implementing a connection retry mechanism. The flag has been enabled by default for 2.3 years with no subsequent changes, indicating stable behavior.

## Justification for Removal

### Stability Evidence
- **Time enabled:** 2.3 years in production across all deployments
- **Toggle history:** Single clean enablement in Release 4.1, never reverted
- **Issue history:** No known issues or customer complaints requiring the old behavior
- **Dependency status:** Works in conjunction with other sensor features but is independently stable

### Feature Maturity
- The connection retry mechanism is now the expected and documented Sensor behavior
- Removing the fallback to "restart on disconnect" simplifies sensor lifecycle management
- All integration tests now assume retry behavior is enabled

### Risk Factors
- **Medium Risk:** This affects critical Sensor behavior (connection handling)
- **Mitigation:** The "enabled" behavior has been the default for 2+ years
- **Rollback:** Clean revert possible if issues arise (re-add the flag)

## Success Criteria

- [ ] All references to `PreventSensorRestartOnDisconnect` removed
- [ ] Sensor always uses retry behavior on disconnect
- [ ] All tests pass
- [ ] No dead code remains
- [ ] Sensor behavior unchanged from current default

## Files Modified

1. `pkg/features/list.go` - Remove flag definition
2. `sensor/common/sensor/sensor.go` - Simplify Start() and Stop() methods
3. `sensor/tests/connection/runtime/runtime_test.go` - Remove test setup
4. `sensor/tests/connection/k8sreconciliation/reconcile_test.go` - Remove test setup
5. `sensor/tests/connection/alerts/alert_test.go` - Remove test setup

**Total:** 5 files modified
