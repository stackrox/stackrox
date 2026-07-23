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

## 3. referrer-byopki

Sigstore bundle format, BYOPKI (bring-your-own-PKI) certificate chain, attached
as OCI referrer. The signing certificate must carry OIDC extensions for issuer
(`https://testing.org`) and identity (`team-a@testing.org`).

### Generate the CA hierarchy

```bash
# Root CA.
openssl req -x509 -newkey rsa:4096 -keyout root-ca.key -out root-ca.pem \
  -days 10000 -nodes -subj "/C=US/L=Testing/O=DEV/OU=Testing/CN=Testing CA"

# Intermediate CA.
openssl req -newkey rsa:4096 -keyout intermediate-ca.key -out intermediate-ca.csr \
  -nodes -subj "/C=US/L=Testing/O=DEV/OU=Testing/CN=Testing Intermediate CA"
openssl x509 -req -in intermediate-ca.csr -CA root-ca.pem -CAkey root-ca.key \
  -CAcreateserial -out intermediate-ca.pem -days 10000 \
  -extfile <(printf "basicConstraints=critical,CA:TRUE,pathlen:2\nkeyUsage=critical,keyCertSign")

# Leaf signing key and certificate with OIDC extensions.
openssl req -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
  -keyout leaf.key -out leaf.csr -nodes \
  -subj "/C=US/L=Testing/O=DEV/OU=Testing/CN=team-a@testing.org"
openssl x509 -req -in leaf.csr -CA intermediate-ca.pem -CAkey intermediate-ca.key \
  -CAcreateserial -out leaf.pem -days 10000 \
  -extfile <(printf "1.3.6.1.4.1.57264.1.1=ASN1:UTF8String:https://testing.org\n1.3.6.1.4.1.57264.1.7=ASN1:UTF8String:team-a@testing.org")

# CA bundle (intermediate + root, order matters).
cat intermediate-ca.pem root-ca.pem > ca-bundle.pem
```

### Sign the image

```bash
crane cp docker.io/library/alpine:latest \
  quay.io/rhacs-eng/qa-signatures:referrer-byopki

DIGEST=$(crane digest quay.io/rhacs-eng/qa-signatures:referrer-byopki)

cosign sign \
  --key leaf.key \
  --certificate leaf.pem \
  --certificate-chain ca-bundle.pem \
  --new-bundle-format \
  "quay.io/rhacs-eng/qa-signatures@$DIGEST"
```

## After signing

Update `ImageSignatureVerificationTest.groovy` with:
- `REFERRER_COSIGN_PUBLIC_KEY`: contents of `cosign.pub`
- `REFERRER_PUBKEY_MATCHING_IMAGE_DIGEST`: digest from step 1
- `REFERRER_KEYLESS_SIGSTORE_MATCHING_IMAGE_DIGEST`: digest from step 2
- `REFERRER_BYOPKI_IMAGE_DIGEST`: digest from step 3
- `REFERRER_BYOPKI_CA_BUNDLE`: contents of `ca-bundle.pem`

## Verification

```bash
cosign verify --key cosign.pub --new-bundle-format \
  quay.io/rhacs-eng/qa-signatures:referrer-sigstore-bundle-pubkey

cosign verify --new-bundle-format \
  --certificate-identity-regexp=".*@redhat.com" \
  --certificate-oidc-issuer="https://github.com/login/oauth" \
  quay.io/rhacs-eng/qa-signatures:referrer-sigstore-bundle-keyless

cosign verify --new-bundle-format \
  --certificate-identity="team-a@testing.org" \
  --certificate-oidc-issuer="https://testing.org" \
  --cert-chain=ca-bundle.pem \
  quay.io/rhacs-eng/qa-signatures:referrer-byopki
```
