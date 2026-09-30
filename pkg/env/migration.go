package env

// UnsafeAllowUnsupportedUpgrade bypasses only the product-version upgrade gate.
// It does not restore pruned migrations or override sequence-based rollback safety.
var UnsafeAllowUnsupportedUpgrade = RegisterBooleanSetting("ROX_UNSAFE_ALLOW_UNSUPPORTED_UPGRADE", false)
