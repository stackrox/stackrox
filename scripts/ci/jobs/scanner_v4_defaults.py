"""Scanner V4 CI defaults. Intended for CI use only; do not use in production."""

# Restricts which vulnerability bundles are loaded by the Scanner V4 matcher.
VULN_BUNDLE_ALLOWLIST = "alpine,debian,epss,manual,nvd,osv,rhel-vex,stackrox-rhel-csaf,ubuntu"
