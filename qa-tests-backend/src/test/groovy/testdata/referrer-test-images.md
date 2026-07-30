# Referrer-based Signature Test Images

Instructions for creating test images with OCI 1.1 referrer-based sigstore
bundle signatures for `ImageSignatureVerificationTest`.

## Prerequisites

```bash
cosign version  # requires v3+ for --new-bundle-format
crane version   # for copying base images
```

## Generate a key pair

```bash
cosign generate-key-pair
# Produces cosign.key (private) and cosign.pub (public).
# Store both in Bitwarden alongside existing signature test credentials.
```

## 1. referrer-sigstore-bundle-pubkey

Sigstore bundle format, public key, attached as OCI referrer.

```bash
crane cp docker.io/library/alpine:latest \
  quay.io/rhacs-eng/qa-signatures:referrer-sigstore-bundle-pubkey

DIGEST=$(crane digest quay.io/rhacs-eng/qa-signatures:referrer-sigstore-bundle-pubkey)

cosign sign \
  --key cosign.key \
  --new-bundle-format \
  "quay.io/rhacs-eng/qa-signatures@$DIGEST"
```

## 2. referrer-sigstore-bundle-keyless

Sigstore bundle format, keyless (public Sigstore), attached as OCI referrer.
Sign with an `@redhat.com` GitHub identity to match the test's OIDC config.

```bash
crane cp docker.io/library/alpine:latest \
  quay.io/rhacs-eng/qa-signatures:referrer-sigstore-bundle-keyless

DIGEST=$(crane digest quay.io/rhacs-eng/qa-signatures:referrer-sigstore-bundle-keyless)

cosign sign \
  --new-bundle-format \
  "quay.io/rhacs-eng/qa-signatures@$DIGEST"
```

## After signing

Update `ImageSignatureVerificationTest.groovy` with:
- `REFERRER_COSIGN_PUBLIC_KEY`: contents of `cosign.pub`
- `REFERRER_SIGSTORE_BUNDLE_PUBKEY_IMAGE_DIGEST`: digest from step 1
- `REFERRER_SIGSTORE_BUNDLE_KEYLESS_IMAGE_DIGEST`: digest from step 2

## Verification

```bash
cosign verify --key cosign.pub --new-bundle-format \
  quay.io/rhacs-eng/qa-signatures:referrer-sigstore-bundle-pubkey

cosign verify --new-bundle-format \
  --certificate-identity-regexp=".*@redhat.com" \
  --certificate-oidc-issuer="https://github.com/login/oauth" \
  quay.io/rhacs-eng/qa-signatures:referrer-sigstore-bundle-keyless
```
