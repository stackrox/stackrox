# Splunk + StackRox TA image used by the Splunk E2E test (qa-tests-backend).
# This is a pin file, not a build input: nothing builds or pushes it. It exists so
# Dependabot keeps the digest fresh. SplunkUtil.groovy reads the reference below at
# runtime, so the E2E test always runs against the exact, committed digest.
# Dependabot auto-bumps this digest via the '.github/test-images' docker ecosystem
# config (see .github/dependabot.yaml). Because quay.io/rhacs-eng/qa is private,
# Dependabot authenticates using the 'quay-rhacs-eng' registry defined there.
FROM quay.io/rhacs-eng/qa:splunk-ta-test-latest@sha256:ac8cef11254acbed5bb32bc30062a712f8ca956bc78f6deca26edd74742e16d1
