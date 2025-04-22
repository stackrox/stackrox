import static Services.checkForNoActiveViolations
import static Services.waitForViolation
import static util.Helpers.withRetry

import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.ScopeOuterClass
import io.stackrox.proto.storage.SignatureIntegrationOuterClass.CertificateTransparencyLogVerification
import io.stackrox.proto.storage.SignatureIntegrationOuterClass.CosignPublicKeyVerification
import io.stackrox.proto.storage.SignatureIntegrationOuterClass.CosignCertificateVerification
import io.stackrox.proto.storage.SignatureIntegrationOuterClass.SignatureIntegration
import io.stackrox.proto.storage.SignatureIntegrationOuterClass.TransparencyLogVerification

import objects.Deployment
import services.ImageService
import services.PolicyService
import services.SignatureIntegrationService
import services.CertificateVerificationArgs
import services.TransparencyLogVerificationArgs

import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll

class ImageSignatureVerificationTest extends BaseSpecification {

    static final private String SIGNATURE_TESTING_NAMESPACE = "qa-signature-tests"

    // Names of the signature integration + policies that use the integration as the value of Trusted image signers.
    static final private String DISTROLESS = "Distroless"
    static final private String TEKTON = "Tekton"
    static final private String UNVERIFIABLE = "Unverifiable"
    static final private String DISTROLESS_AND_TEKTON = "Distroless+Tekton"
    static final private String POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE = "Distroless+Tekton+Unverifiable"
    static final private String SAME_DIGEST = "Same+Digest"
    static final private String BYOPKI_WILDCARD = "BYOPKI-Wildcard"
    static final private String BYOPKI_UNVERIFIABLE = "BYOPKI-Unverifiable"
    static final private String BYOPKI_MATCHING = "BYOPKI-Matching"
    static final private String BYOPKI_WILDCARD_AND_TEKTON = "BOYPKI-Wildcard+Tekton"
    static final private String KEYLESS_SIGSTORE_MATCHING = "Keyless-Sigstore-Matching"
    static final private String KEYLESS_SIGSTORE_UNVERIFIABLE = "Keyless-Sigstore-Unverifiable"

    // List of integration names used within tests.
    // NOTE: If you add a new name, make sure to add it here.
    static final private List<String> INTEGRATION_NAMES = [
            DISTROLESS,
            TEKTON,
            UNVERIFIABLE,
            DISTROLESS_AND_TEKTON,
            SAME_DIGEST,
            BYOPKI_WILDCARD,
            BYOPKI_UNVERIFIABLE,
            BYOPKI_MATCHING,
            BYOPKI_WILDCARD_AND_TEKTON,
            KEYLESS_SIGSTORE_MATCHING,
            KEYLESS_SIGSTORE_UNVERIFIABLE,
    ]


    // Public keys used within signature integrations.
    static final private NO_PUBLIC_KEYS = [:]
    static final private Map<String, String> DISTROLESS_PUBLIC_KEY = [
            // Source: https://vault.bitwarden.com/#/vault?itemId=95313e19-de46-4533-b160-af620120452a.
            "Distroless": """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEcVdWNZ/4iPmE7xbpqO4TceXQh6Wy
8Vgkra4Ip0w+HmHYNcv5yQELuuCF+5GpNfnFy997OUivUXEXb/gButu0qQ==
-----END PUBLIC KEY-----""",
    ]
    static final private Map<String, String> TEKTON_COSIGN_PUBLIC_KEY = [
            // Source: https://vault.bitwarden.com/#/vault?itemId=95313e19-de46-4533-b160-af620120452a.
            "Tekton": """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5iePLkmv6t286ufeqp6HLZ9T9wry
bXlAKIPDApWJ4LY9QBESP4xed+CsLkm1ErLFJXpp+AB2YpqP8KYpvAp3Xg==
-----END PUBLIC KEY-----""",
    ]
    static final private Map<String, String> UNVERIFIABLE_COSIGN_PUBLIC_KEY = [
            // Manually created cosing public key via `cosign generate-key-pair`
            // does not verify UNVERIFIABLE_DEPLOYMENT
            "Unverifiable": """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEUpphKrUYSHvrR+r82Jn7Evg/d3L9
w9e2Azq1OYIh/pbeBMHARDrBaqqmuMR9+BfAaPAYdkNTU6f58M2zBbuL0A==
-----END PUBLIC KEY-----""",
    ]
    static final private Map<String, String> SAME_DIGEST_COSIGN_PUBLIC_KEY = [
            // Source: https://vault.bitwarden.com/#/vault?itemId=95313e19-de46-4533-b160-af620120452a.
            "Docker": """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEhsRRb4sl0Y4PeVSk9w/eYaWwigXj
QC+pUMTUP/ZmrvmKaA+pi55F+w3LqVJ17zwXKjaOEiEpn/+lntl/ieweeQ==
-----END PUBLIC KEY-----""",
    ]

    // Root certificate used within signature integration.
    // Source: https://vault.bitwarden.com/#/vault?itemId=8551a79b-e774-48bd-bc14-b19a0047c850.
    static final private String BYOPKI_ROOT_CA = """\
-----BEGIN CERTIFICATE-----
MIIFizCCA3OgAwIBAgIUKUP5Gi0K7c0FwuJ8otDaPaXa5fYwDQYJKoZIhvcNAQEL
BQAwVDELMAkGA1UEBhMCVVMxEDAOBgNVBAcMB1Rlc3RpbmcxDDAKBgNVBAoMA0RF
VjEQMA4GA1UECwwHVGVzdGluZzETMBEGA1UEAwwKVGVzdGluZyBDQTAgFw0yNDA2
MjUwNDE3NDBaGA8yMDUxMTExMDA0MTc0MFowVDELMAkGA1UEBhMCVVMxEDAOBgNV
BAcMB1Rlc3RpbmcxDDAKBgNVBAoMA0RFVjEQMA4GA1UECwwHVGVzdGluZzETMBEG
A1UEAwwKVGVzdGluZyBDQTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIB
AMQvEO69CS+dM3bvtFdYxr5imHMLZVyy5drVAFvLKD9SaZzjPQ+VKtZ0zB9TUJrh
x6sKTlBk+pFg7iY0BYr+FCWRw2QqhT+4fFST2HGlRzSexw7Ah8E67QH+p+1Qypel
/ae6/4KaH6ijUY5iqySEhMHBVcAdD+s++kbTDQ4MsZCYbt+zdE8KgkI68kAJ6Fpv
pFw+cFZI5wg8thAR+j1U2rfD8O0E9w0xX+2+iIyHPsfFVN5oyK140Wg43ApZQu7e
FPLjRa22fFbsgoijouj/74FqINWcxc+ZRzv9HNIwnWt/mBfE0nNvoeqYQFr9nLGu
O/ZUR954IU+br+OAihMrDgAWU5qSFdgXeITTvxSuN+F+3HGmEQWFo+oP/gYAMM28
ZtgiKWtOZigZhn7b1sdl0XMOhaR+wHqPvHRYfqXrdz3ektHA9KlXV/8WzTXLWoUV
snpCqJY9i193X+WsVMbF3sp+Zmdu2QXEmJtNkUd6zvwuZ4S4JyePF5DXoAq2IsoM
+4A4HYisUgAS2ZrWPpvCQ9Z31CO0aMKSgHPSsU1Q9QEg17V420vvg5bqn8Zbedyb
nzmJ4Vp2Ccc6YIb7QWdr95UJ7JMoNzLzFLoVCa3XFDkeT2stZlzCrKMBhaVYIewr
OOlTg67Sa77H7JUYfvmE+i9fxqvfs3VZ5S+w4CDy7rLXAgMBAAGjUzBRMB0GA1Ud
DgQWBBSPppE/33XnVV+YBvL+osCzuD1lHzAfBgNVHSMEGDAWgBSPppE/33XnVV+Y
BvL+osCzuD1lHzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4ICAQCm
sTJmiwRX6w0kM/erc8aMdCOcIW4ofSfNkQMdaW/S9pwgnipEF8eKvvkblKvrN+kf
CntWIOGqOgaxkWlC0yOTqKwzN162mjvRaflUCEDkwLhjqFsgn1g1coma3AScBDMf
HnpghLH3AAtxs/eDsbvJa70Ybg/3l2oHTLe6kxsVs5k/zqmvIIxOmSqvNFa/5kGg
iWBz/1X/mqc2lD5/cgz3UrM7ZMr1r4IhGK095ljKiCA1fTrlQrgSQe/3pGm1ShjP
mn5LbCiIr+JCrNFRV9ivSL88mLEJqbWx+qLQhwde8C09c7EGxQZKHfCI/QcdE3EG
UYyRYQzXS4bgpMRBw+l2saGjYWd1EBUl5kj3rCDmtT4u2kln9SwqYMT9ZJ/PMtTQ
WKQj6qVYXla6XLmED8XWqMY7dDKHoy0XLwgUS5SnD+D68Fqfpt4kzm/RnuuBmQl8
6+BO9NMNMo0+TOm9ItEouE/GD4TbRu59h0Q+usyidmUF79Cj+6qKLW//KRQyrEKa
LgH/JNuwVgUip0EnN1wsVXSCVpCIoKKANRkQSIfKJwRzlD0JO5moL225W5gp68Zx
p6K21vDtW3Cmofa6rpV+ZTd60iyTv5YhVd/h/Gunfm2zLhZKb75aJWpEzwn51QvK
nzTe7BpOmVwmqLkIefEJe5L4PSXtp2KFLZqGO/kY5A==
-----END CERTIFICATE-----"""
    static final private String BYOPKI_WILDCARD_ISSUER = ".*"
    static final private String BYOPKI_WILDCARD_IDENTITY = ".*"
    static final private String BYOPKI_MATCHING_ISSUER = "https://testing.org"
    static final private String BYOPKI_MATCHING_IDENTITY = "team-a@testing.org"
    static final private String BYOPKI_UNVERIFIABLE_ISSUER = "invalid"
    static final private String BYOPKI_UNVERIFIABLE_IDENTITY = "invalid"

    static final private String KEYLESS_SIGSTORE_ISSUER = "https://github.com/login/oauth"
    static final private String KEYLESS_SIGSTORE_IDENTITY = ".*@redhat.com"

    static final private String DISTROLESS_IMAGE_DIGEST =
            "sha256:bc217643f9c04fc8131878d6440dd88cf4444385d45bb25995c8051c29687766"
    static final private String TEKTON_IMAGE_DIGEST =
            "sha256:d12d420438235ccee3f4fcb72cf9e5b2b79f60f713595fe1ada254d10167dfc6"
    static final private String UNVERIFIABLE_IMAGE_DIGEST =
            "sha256:743cf31b5c29c227aa1371eddd9f9313b2a0487f39ccfc03ec5c89a692c4a0c7"
    static final private String WITHOUT_SIGNATURE_IMAGE_DIGEST =
            "sha256:b73f527d86e3461fd652f62cf47e7b375196063bbbd503e853af5be16597cb2e"
    static final private String SAME_DIGEST_NO_SIGNATURE_IMAGE_DIGEST =
            "sha256:dd2d0ac3fff2f007d99e033b64854be0941e19a2ad51f174d9240dda20d9f534"
    static final private String SAME_DIGEST_WITH_SIGNATURE_IMAGE_DIGEST =
            "sha256:dd2d0ac3fff2f007d99e033b64854be0941e19a2ad51f174d9240dda20d9f534"
    static final private String BYOPKI_IMAGE_DIGEST =
            "sha256:7b3ccabffc97de872a30dfd234fd972a66d247c8cfc69b0550f276481852627c"
    static final private String KEYLESS_SIGSTORE_IMAGE_DIGEST =
            "sha256:37f7b378a29ceb4c551b1b5582e27747b855bbfaa73fa11914fe0df028dc581f"

    static final private List<String> IMAGE_DIGESTS = [
            DISTROLESS_IMAGE_DIGEST,
            TEKTON_IMAGE_DIGEST,
            UNVERIFIABLE_IMAGE_DIGEST,
            WITHOUT_SIGNATURE_IMAGE_DIGEST,
            SAME_DIGEST_NO_SIGNATURE_IMAGE_DIGEST,
            SAME_DIGEST_WITH_SIGNATURE_IMAGE_DIGEST,
            BYOPKI_IMAGE_DIGEST,
            KEYLESS_SIGSTORE_IMAGE_DIGEST,
    ]

    // Deployment holding an image which has a cosign signature that is verifiable with the DISTROLESS_PUBLIC_KEY.
    static final private Deployment DISTROLESS_DEPLOYMENT = new Deployment()
            .setName("with-signature-verified-by-distroless")
            // quay.io/rhacs-eng/qa-signatures:distroless-base-multiarch
            .setImage("quay.io/rhacs-eng/qa-signatures:distroless-base-multiarch@$DISTROLESS_IMAGE_DIGEST")
            .addLabel("app", "image-with-signature-distroless-test")
            .setCommand(["sleep", "6000"])
            .setNamespace(SIGNATURE_TESTING_NAMESPACE)

    // Deployment holding an image which has a cosign signature that is verifiable with the TEKTON_PUBLIC_KEY.
    static final private Deployment TEKTON_DEPLOYMENT = new Deployment()
            .setName("with-signature-verified-by-tekton")
            // quay.io/rhacs-eng/qa-signatures:tekton-multiarch
            .setImage("quay.io/rhacs-eng/qa-signatures:tekton-multiarch@$TEKTON_IMAGE_DIGEST")
            .addLabel("app", "image-with-signature-tekton-test")
            .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
            .setNamespace(SIGNATURE_TESTING_NAMESPACE)

    // Deployment holding an image which has a cosign signature that is not verifiable by any cosign public key.
    static final private Deployment UNVERIFIABLE_DEPLOYMENT = new Deployment()
            .setName("with-signature-unverifiable")
            // quay.io/rhacs-eng/qa-signatures:centos9-multiarch
            .setImage("quay.io/rhacs-eng/qa-signatures:centos9-multiarch@$UNVERIFIABLE_IMAGE_DIGEST")
            .addLabel("app", "image-with-unverifiable-signature-test")
            .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
            .setNamespace(SIGNATURE_TESTING_NAMESPACE)

    // Deployment holding an image which does not have a cosign signature.
    static final private Deployment WITHOUT_SIGNATURE_DEPLOYMENT = new Deployment()
            .setName("without-signature")
            // quay.io/rhacs-eng/qa-multi-arch:nginx-204a9a8
            .setImage("quay.io/rhacs-eng/qa-multi-arch@$WITHOUT_SIGNATURE_IMAGE_DIGEST")
            .addLabel("app", "image-without-signature")
            .setNamespace(SIGNATURE_TESTING_NAMESPACE)

    // Deployment holding an image with the same digest as quay.io/rhacs-eng/qa-signatures:nginx that does
    // not have a cosign signature associated with it.
    static final private Deployment SAME_DIGEST_NO_SIGNATURE = new Deployment()
            .setName("same-digest-without-signature")
            // quay.io/rhacs-eng/qa--multi-arch:enforcement
            .setImage("quay.io/rhacs-eng/qa-multi-arch@$SAME_DIGEST_NO_SIGNATURE_IMAGE_DIGEST")
            .addLabel("app", "image-same-digest-without-signature")
            .setNamespace(SIGNATURE_TESTING_NAMESPACE)

    // Deployment holding an image with the same digest as quay.io/rhacs-eng/qa:enforcement that does
    // have a cosign signature associated with it.
    static final private Deployment SAME_DIGEST_WITH_SIGNATURE = new Deployment()
            .setName("same-digest-with-signature")
            // quay.io/rhacs-eng/qa-signatures:nginx-multiarch
            .setImage("quay.io/rhacs-eng/qa-signatures:nginx-multiarch@$SAME_DIGEST_WITH_SIGNATURE_IMAGE_DIGEST")
            .addLabel("app", "image-same-digest-with-signature")
            .setNamespace(SIGNATURE_TESTING_NAMESPACE)

    // Deployment holding an image with BYOPKI. BYOPKI means that the image signature
    // also has a certificate and certificate chain attached, which can be used to verify the signature.
    static final private Deployment BYOPKI_DEPLOYMENT = new Deployment()
            .setName("byopki")
            .setImage("quay.io/rhacs-eng/qa-signatures:byopki@$BYOPKI_IMAGE_DIGEST")
            .addLabel("app", "image-with-byopki")
            .setCommand(["sleep", "600"])
            .setNamespace(SIGNATURE_TESTING_NAMESPACE)

    // Deployment holding an image signed by keyless Cosign using public Sigstore instances.
    static final private Deployment KEYLESS_SIGSTORE_DEPLOYMENT = new Deployment()
            .setName("keyless-sigstore")
            .setImage("quay.io/rhacs-eng/qa-signatures:keyless-sigstore@$KEYLESS_SIGSTORE_IMAGE_DIGEST")
            .addLabel("app", "image-with-keyless-sigstore")
            .setCommand(["sleep", "600"])
            .setNamespace(SIGNATURE_TESTING_NAMESPACE)

    // List of deployments used within the tests. This will be used during setup of the spec / teardown to create /
    // delete all deployments.
    // NOTE: If you add another deployment, make sure to add it here as well.
    static final private List<Deployment> DEPLOYMENTS = [
            DISTROLESS_DEPLOYMENT,
            TEKTON_DEPLOYMENT,
            UNVERIFIABLE_DEPLOYMENT,
            WITHOUT_SIGNATURE_DEPLOYMENT,
            SAME_DIGEST_NO_SIGNATURE,
            SAME_DIGEST_WITH_SIGNATURE,
            BYOPKI_DEPLOYMENT,
            KEYLESS_SIGSTORE_DEPLOYMENT,
    ]

    // Base policy which will be used for creating subsequent policies that have signature integration IDs as values.
    static final private Policy.Builder BASE_POLICY = Policy.newBuilder()
            .addLifecycleStages(PolicyOuterClass.LifecycleStage.DEPLOY)
            .addCategories("Test")
            .setDisabled(false)
            .setSeverityValue(2)
            .addAllScope([SIGNATURE_TESTING_NAMESPACE].collect
                    { ScopeOuterClass.Scope.newBuilder().setNamespace(it).build() })

    @Shared
    static final private List<String> CREATED_POLICY_IDS = []

    @Shared
    static final private Map<String, String> CREATED_SIGNATURE_INTEGRATIONS = [:]

    def setupSpec() {
        orchestrator.createNamespace(SIGNATURE_TESTING_NAMESPACE)
        addStackroxImagePullSecret(SIGNATURE_TESTING_NAMESPACE)

        // Signature integration "Distroless" which holds only the distroless cosign public key.
        String distrolessSignatureIntegrationID = createSignatureIntegration(
            DISTROLESS, DISTROLESS_PUBLIC_KEY
        )
        assert distrolessSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(DISTROLESS, distrolessSignatureIntegrationID)

        // Signature integration "Tekton" which holds only the tekton cosign public key.
        String tektonSignatureIntegrationID = createSignatureIntegration(
            TEKTON, TEKTON_COSIGN_PUBLIC_KEY
        )
        assert tektonSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(TEKTON, tektonSignatureIntegrationID)

        // Signature integration "Unverifiable" which holds only the unverifiable cosign public key.
        String unverifiableSignatureIntegrationID = createSignatureIntegration(
            UNVERIFIABLE, UNVERIFIABLE_COSIGN_PUBLIC_KEY
        )
        assert unverifiableSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(UNVERIFIABLE, unverifiableSignatureIntegrationID)

        // Signature integration "Same+Digest" which holds only the same digest cosign public key.
        String sameDigestSignatureIntegrationID = createSignatureIntegration(
            SAME_DIGEST, SAME_DIGEST_COSIGN_PUBLIC_KEY
        )
        assert sameDigestSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(SAME_DIGEST, sameDigestSignatureIntegrationID)

        // Signature integration "BYOPKI_WILDCARD" which holds the root CA + wildcard regex for
        // issuer and identity.
        String byopkiWildcardSignatureIntegrationID = createSignatureIntegration(
            BYOPKI_WILDCARD, NO_PUBLIC_KEYS,
            [chain: BYOPKI_ROOT_CA, identity: BYOPKI_WILDCARD_IDENTITY, issuer: BYOPKI_WILDCARD_ISSUER]
        )
        assert byopkiWildcardSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(BYOPKI_WILDCARD, byopkiWildcardSignatureIntegrationID)

        // Signature integration "BYOPKI_MATCHING" which holds the root CA + matching identity
        // and issuer.
        String byopkiMatchingSignatureIntegrationID = createSignatureIntegration(
            BYOPKI_MATCHING, NO_PUBLIC_KEYS,
            [chain: BYOPKI_ROOT_CA, identity: BYOPKI_MATCHING_IDENTITY, issuer: BYOPKI_MATCHING_ISSUER]
        )
        assert byopkiMatchingSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(BYOPKI_MATCHING, byopkiMatchingSignatureIntegrationID)

        // Signature integration "BYOPKI_UNVERIFIABLE" which holds the root CA + a non-matching
        // identity and issuer.
        String byopkiUnverifiableSignatureIntegrationID = createSignatureIntegration(
            BYOPKI_UNVERIFIABLE, NO_PUBLIC_KEYS,
            [chain: BYOPKI_ROOT_CA, identity: BYOPKI_UNVERIFIABLE_IDENTITY, issuer: BYOPKI_UNVERIFIABLE_ISSUER]
        )
        assert byopkiUnverifiableSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(BYOPKI_UNVERIFIABLE, byopkiUnverifiableSignatureIntegrationID)

        // Signature integration "Distroless+Tekton" which holds both distroless and tekton cosign public keys.
        Map<String,String> mergedKeys = DISTROLESS_PUBLIC_KEY.clone() as Map<String, String>
        mergedKeys.putAll(TEKTON_COSIGN_PUBLIC_KEY.entrySet())
        String distrolessAndTektonSignatureIntegrationID = createSignatureIntegration(
                DISTROLESS_AND_TEKTON, mergedKeys
        )
        assert distrolessAndTektonSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(DISTROLESS_AND_TEKTON, distrolessAndTektonSignatureIntegrationID)

        // Signature integartion "BYOPKI-Wildcard+Tekton" which holds both BYOPKI wildcard and Tekton.
        String byopkiWildcardAndTektonSignatureIntegrationID = createSignatureIntegration(
            BYOPKI_WILDCARD_AND_TEKTON, TEKTON_COSIGN_PUBLIC_KEY,
            [chain: BYOPKI_ROOT_CA, identity: BYOPKI_WILDCARD_IDENTITY, issuer: BYOPKI_WILDCARD_ISSUER]
        )
        assert byopkiWildcardAndTektonSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(BYOPKI_WILDCARD_AND_TEKTON, byopkiWildcardAndTektonSignatureIntegrationID)

        // Signature integration "Keyless-Sigstore-Matching" which holds the default Sigstore CAs
        // and enables transparency log validation.
        String keylessSigstoreMatchingSignatureIntegrationID = createSignatureIntegration(
            KEYLESS_SIGSTORE_MATCHING, NO_PUBLIC_KEYS,
            [chain: "", identity: KEYLESS_SIGSTORE_IDENTITY, issuer: KEYLESS_SIGSTORE_ISSUER,
            ctlogEnabled: true],
            [enabled: true, url: "https://rekor.sigstore.dev"]
        )
        assert keylessSigstoreMatchingSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(KEYLESS_SIGSTORE_MATCHING, keylessSigstoreMatchingSignatureIntegrationID)

        // Signature integration "Keyless-Sigstore-Unverifiable" which holds the default Sigstore CAs
        // and disables transparency log validation. Verification must fail because the certificate issued
        // by Fulcio has expired and can only be verified with the timestamp from the transparency log entry.
        String keylessSigstoreUnverifiableSignatureIntegrationID = createSignatureIntegration(
            KEYLESS_SIGSTORE_UNVERIFIABLE, NO_PUBLIC_KEYS,
            [chain: "", identity: KEYLESS_SIGSTORE_IDENTITY, issuer: KEYLESS_SIGSTORE_ISSUER,
            ctlogEnabled: true],
            [enabled: false]
        )
        assert keylessSigstoreUnverifiableSignatureIntegrationID
        CREATED_SIGNATURE_INTEGRATIONS.put(KEYLESS_SIGSTORE_UNVERIFIABLE,
            keylessSigstoreUnverifiableSignatureIntegrationID)

        // Create all required deployments.
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        DEPLOYMENTS.each { assert Services.waitForDeployment(it) }

        // Wait until we received metadata from all images we want to test. This will ensure that enrichment
        // has finalized.
        withRetry(20, 30) {
            for (digest in IMAGE_DIGESTS) {
                ImageOuterClass.Image image = ImageService.getImage(digest, false)
                assert image
                assert !image.getNotesList().contains(ImageOuterClass.Image.Note.MISSING_METADATA)
                assert !image.getNotPullable()
            }
        }

        // Create the policy builders using the signature integration IDs.
        List<Policy.Builder> policyBuilders = []
        for (integrationName in INTEGRATION_NAMES) {
            Policy.Builder builder = createPolicyBuilderWithSignatureCriteria(integrationName,
                    [CREATED_SIGNATURE_INTEGRATIONS.get(integrationName, "")])
            assert builder
            policyBuilders.add(builder)
        }

        // Create a policy which holds three signature integrations.
        Policy.Builder builder = createPolicyBuilderWithSignatureCriteria(POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE,
        [CREATED_SIGNATURE_INTEGRATIONS.get(DISTROLESS), CREATED_SIGNATURE_INTEGRATIONS.get(TEKTON),
         CREATED_SIGNATURE_INTEGRATIONS.get(UNVERIFIABLE)])
        assert builder
        policyBuilders.add(builder)

        // Create policies we use within tests.
        for (policyBuilder in policyBuilders) {
            Policy policy = policyBuilder.build()
            String policyID = PolicyService.createNewPolicy(policy)
            assert policyID
            CREATED_POLICY_IDS.add(policyID)
        }

        // Reassessing policies will trigger a re-enrichment of images.
        PolicyService.reassessPolicies()
    }

    def cleanupSpec() {
        // Delete all deployments.
        DEPLOYMENTS.each { orchestrator.deleteAndWaitForDeploymentDeletion(it) }

        // Delete all created policies.
        CREATED_POLICY_IDS.each { PolicyService.deletePolicy(it) }

        // Delete all created signature integrations.
        CREATED_SIGNATURE_INTEGRATIONS.each
                { SignatureIntegrationService.deleteSignatureIntegration(it.value) }

        orchestrator.deleteNamespace(SIGNATURE_TESTING_NAMESPACE)
        orchestrator.waitForNamespaceDeletion(SIGNATURE_TESTING_NAMESPACE)
    }

    @Unroll
    @SuppressWarnings('LineLength')
    @Tag("BAT")
    @Tag("Integration")
    @Tag("PZ")
    def "Check violations of policy '#policyName' for deployment '#deployment.name'"() {
        expect:
        "Verify deployment has expected violations"
        if (expectViolations) {
            assert waitForViolation(deployment.name, policyName)
        } else {
            assert checkForNoActiveViolations(deployment.name, policyName, 60)
        }

        where:
        policyName                                 | deployment                   | expectViolations
        // Distroless should create alerts for all deployments except those using distroless images.
        DISTROLESS                                 | BYOPKI_DEPLOYMENT            | true
        DISTROLESS                                 | DISTROLESS_DEPLOYMENT        | false
        DISTROLESS                                 | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        DISTROLESS                                 | SAME_DIGEST_NO_SIGNATURE     | true
        DISTROLESS                                 | SAME_DIGEST_WITH_SIGNATURE   | true
        DISTROLESS                                 | TEKTON_DEPLOYMENT            | true
        DISTROLESS                                 | UNVERIFIABLE_DEPLOYMENT      | true
        DISTROLESS                                 | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // Tekton should create alerts for all deployments except those using tekton images.
        TEKTON                                     | BYOPKI_DEPLOYMENT            | true
        TEKTON                                     | DISTROLESS_DEPLOYMENT        | true
        TEKTON                                     | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        TEKTON                                     | SAME_DIGEST_NO_SIGNATURE     | true
        TEKTON                                     | SAME_DIGEST_WITH_SIGNATURE   | true
        TEKTON                                     | TEKTON_DEPLOYMENT            | false
        TEKTON                                     | UNVERIFIABLE_DEPLOYMENT      | true
        TEKTON                                     | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // Unverifiable should create alerts for all deployments.
        UNVERIFIABLE                               | BYOPKI_DEPLOYMENT            | true
        UNVERIFIABLE                               | DISTROLESS_DEPLOYMENT        | true
        UNVERIFIABLE                               | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        UNVERIFIABLE                               | SAME_DIGEST_NO_SIGNATURE     | true
        UNVERIFIABLE                               | SAME_DIGEST_WITH_SIGNATURE   | true
        UNVERIFIABLE                               | TEKTON_DEPLOYMENT            | true
        UNVERIFIABLE                               | UNVERIFIABLE_DEPLOYMENT      | true
        UNVERIFIABLE                               | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // Distroless and tekton should create alerts for all deployments except those using distroless / tekton images.
        DISTROLESS_AND_TEKTON                      | BYOPKI_DEPLOYMENT            | true
        DISTROLESS_AND_TEKTON                      | DISTROLESS_DEPLOYMENT        | false
        DISTROLESS_AND_TEKTON                      | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        DISTROLESS_AND_TEKTON                      | SAME_DIGEST_NO_SIGNATURE     | true
        DISTROLESS_AND_TEKTON                      | SAME_DIGEST_WITH_SIGNATURE   | true
        DISTROLESS_AND_TEKTON                      | TEKTON_DEPLOYMENT            | false
        DISTROLESS_AND_TEKTON                      | UNVERIFIABLE_DEPLOYMENT      | true
        DISTROLESS_AND_TEKTON                      | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // Policy with all three integrations should create alerts for all deployments except those using distroless /
        // tekton images.
        POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE | BYOPKI_DEPLOYMENT            | true
        POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE | DISTROLESS_DEPLOYMENT        | false
        POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE | SAME_DIGEST_NO_SIGNATURE     | true
        POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE | SAME_DIGEST_WITH_SIGNATURE   | true
        POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE | TEKTON_DEPLOYMENT            | false
        POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE | UNVERIFIABLE_DEPLOYMENT      | true
        POLICY_WITH_DISTROLESS_TEKTON_UNVERIFIABLE | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // Same digest should create alerts for all deployments except those using alt-nginx image.
        SAME_DIGEST                                | BYOPKI_DEPLOYMENT            | true
        SAME_DIGEST                                | DISTROLESS_DEPLOYMENT        | true
        SAME_DIGEST                                | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        SAME_DIGEST                                | SAME_DIGEST_NO_SIGNATURE     | true
        SAME_DIGEST                                | SAME_DIGEST_WITH_SIGNATURE   | false
        SAME_DIGEST                                | TEKTON_DEPLOYMENT            | true
        SAME_DIGEST                                | UNVERIFIABLE_DEPLOYMENT      | true
        SAME_DIGEST                                | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // BYOPKI wildcard should create alerts for all deployments except the BYOPKI deployment one.
        BYOPKI_WILDCARD                            | BYOPKI_DEPLOYMENT            | false
        BYOPKI_WILDCARD                            | DISTROLESS_DEPLOYMENT        | true
        BYOPKI_WILDCARD                            | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        BYOPKI_WILDCARD                            | SAME_DIGEST_NO_SIGNATURE     | true
        BYOPKI_WILDCARD                            | SAME_DIGEST_WITH_SIGNATURE   | true
        BYOPKI_WILDCARD                            | TEKTON_DEPLOYMENT            | true
        BYOPKI_WILDCARD                            | UNVERIFIABLE_DEPLOYMENT      | true
        BYOPKI_WILDCARD                            | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // BYOPKI matching should create alerts for all deployments except the BYOPKI deployment one.
        BYOPKI_MATCHING                            | BYOPKI_DEPLOYMENT            | false
        BYOPKI_MATCHING                            | DISTROLESS_DEPLOYMENT        | true
        BYOPKI_MATCHING                            | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        BYOPKI_MATCHING                            | SAME_DIGEST_NO_SIGNATURE     | true
        BYOPKI_MATCHING                            | SAME_DIGEST_WITH_SIGNATURE   | true
        BYOPKI_MATCHING                            | TEKTON_DEPLOYMENT            | true
        BYOPKI_MATCHING                            | UNVERIFIABLE_DEPLOYMENT      | true
        BYOPKI_MATCHING                            | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // BYOPKI unverifiable should create alerts for all deployments.
        BYOPKI_UNVERIFIABLE                        | BYOPKI_DEPLOYMENT            | true
        BYOPKI_UNVERIFIABLE                        | DISTROLESS_DEPLOYMENT        | true
        BYOPKI_UNVERIFIABLE                        | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        BYOPKI_UNVERIFIABLE                        | SAME_DIGEST_NO_SIGNATURE     | true
        BYOPKI_UNVERIFIABLE                        | SAME_DIGEST_WITH_SIGNATURE   | true
        BYOPKI_UNVERIFIABLE                        | TEKTON_DEPLOYMENT            | true
        BYOPKI_UNVERIFIABLE                        | UNVERIFIABLE_DEPLOYMENT      | true
        BYOPKI_UNVERIFIABLE                        | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // BYOPKI wildcard + Tekton should create alerts for all deployments except the
        // BYOPKI one and the Tekton one.
        BYOPKI_WILDCARD_AND_TEKTON                 | BYOPKI_DEPLOYMENT            | false
        BYOPKI_WILDCARD_AND_TEKTON                 | DISTROLESS_DEPLOYMENT        | true
        BYOPKI_WILDCARD_AND_TEKTON                 | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        BYOPKI_WILDCARD_AND_TEKTON                 | SAME_DIGEST_NO_SIGNATURE     | true
        BYOPKI_WILDCARD_AND_TEKTON                 | SAME_DIGEST_WITH_SIGNATURE   | true
        BYOPKI_WILDCARD_AND_TEKTON                 | TEKTON_DEPLOYMENT            | false
        BYOPKI_WILDCARD_AND_TEKTON                 | UNVERIFIABLE_DEPLOYMENT      | true
        BYOPKI_WILDCARD_AND_TEKTON                 | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // Keyless Sigstore matching should create alerts for all deployments except the
        // one running the keyless Sigstore signed image.
        KEYLESS_SIGSTORE_MATCHING                  | BYOPKI_DEPLOYMENT            | true
        KEYLESS_SIGSTORE_MATCHING                  | DISTROLESS_DEPLOYMENT        | true
        KEYLESS_SIGSTORE_MATCHING                  | KEYLESS_SIGSTORE_DEPLOYMENT  | false
        KEYLESS_SIGSTORE_MATCHING                  | SAME_DIGEST_NO_SIGNATURE     | true
        KEYLESS_SIGSTORE_MATCHING                  | SAME_DIGEST_WITH_SIGNATURE   | true
        KEYLESS_SIGSTORE_MATCHING                  | TEKTON_DEPLOYMENT            | true
        KEYLESS_SIGSTORE_MATCHING                  | UNVERIFIABLE_DEPLOYMENT      | true
        KEYLESS_SIGSTORE_MATCHING                  | WITHOUT_SIGNATURE_DEPLOYMENT | true
        // Keyless Sigstore unverifiable should create alerts for all deployments.
        KEYLESS_SIGSTORE_UNVERIFIABLE              | BYOPKI_DEPLOYMENT            | true
        KEYLESS_SIGSTORE_UNVERIFIABLE              | DISTROLESS_DEPLOYMENT        | true
        KEYLESS_SIGSTORE_UNVERIFIABLE              | KEYLESS_SIGSTORE_DEPLOYMENT  | true
        KEYLESS_SIGSTORE_UNVERIFIABLE              | SAME_DIGEST_NO_SIGNATURE     | true
        KEYLESS_SIGSTORE_UNVERIFIABLE              | SAME_DIGEST_WITH_SIGNATURE   | true
        KEYLESS_SIGSTORE_UNVERIFIABLE              | TEKTON_DEPLOYMENT            | true
        KEYLESS_SIGSTORE_UNVERIFIABLE              | UNVERIFIABLE_DEPLOYMENT      | true
        KEYLESS_SIGSTORE_UNVERIFIABLE              | WITHOUT_SIGNATURE_DEPLOYMENT | true
    }

    // Helper which creates a policy builder for a policy which uses the image signature policy criteria.
    private static Policy.Builder createPolicyBuilderWithSignatureCriteria(
            String policyName, List<String> signatureIntegrationIDs) {
        def builder = BASE_POLICY.clone().setName(policyName)
        def policyGroup = PolicyOuterClass.PolicyGroup.newBuilder()
                .setFieldName("Image Signature Verified By")
                .setBooleanOperator(PolicyOuterClass.BooleanOperator.OR)
        policyGroup.addAllValues(
                signatureIntegrationIDs.collect
                        { PolicyOuterClass.PolicyValue.newBuilder().setValue(it).build() })
                .setNegate(false)
                .build()
        def policyBuilder = builder.addPolicySections(
                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(policyGroup.build()).build()
        )
        return policyBuilder
    }

    // Helper to create a signature integration with given name, public keys, chain, identity, and issuer.
    private static String createSignatureIntegration(
            String integrationName,
            Map<String, String> namedPublicKeys,
            CertificateVerificationArgs certVerification = null,
            TransparencyLogVerificationArgs tlogVerification = null) {
        SignatureIntegration.Builder builder = SignatureIntegration.newBuilder()
            .setName(integrationName)

        if (!namedPublicKeys.isEmpty()) {
            List<CosignPublicKeyVerification.PublicKey> publicKeys = namedPublicKeys.collect {
                CosignPublicKeyVerification.PublicKey.newBuilder()
                    .setName(it.key).setPublicKeyPemEnc(it.value)
                    .build()
            }
            builder.setCosign(CosignPublicKeyVerification.newBuilder()
                .addAllPublicKeys(publicKeys)
                .build()
            )
        }

        if (certVerification?.identity && certVerification?.issuer) {
            CosignCertificateVerification verification = CosignCertificateVerification.newBuilder()
                .setCertificateChainPemEnc(certVerification?.chain ?: "")
                .setCertificateIdentity(certVerification?.identity ?: "")
                .setCertificateOidcIssuer(certVerification?.issuer ?: "")
                .setCertificateTransparencyLog(CertificateTransparencyLogVerification.newBuilder()
                    .setEnabled(certVerification?.ctlogEnabled ?: false)
                    .build()
                )
                .build()

            builder.addCosignCertificates(verification)
        }

        builder.setTransparencyLog(TransparencyLogVerification.newBuilder()
            .setEnabled(tlogVerification?.enabled ?: false)
            .setUrl(tlogVerification?.url ?: "")
            .build()
        )

        String signatureIntegrationID = SignatureIntegrationService.createSignatureIntegration(
            builder.build()
        )
        return signatureIntegrationID
    }
}
