import type { SecretType } from 'services/SecretsService';

export const secretTypeLabels: Record<SecretType, string> = Object.freeze({
    UNDETERMINED: 'Undetermined',
    PUBLIC_CERTIFICATE: 'Public certificate',
    CERTIFICATE_REQUEST: 'Certificate request',
    PRIVACY_ENHANCED_MESSAGE: 'Privacy enhanced message',
    OPENSSH_PRIVATE_KEY: 'OpenSSH private key',
    PGP_PRIVATE_KEY: 'PGP private key',
    EC_PRIVATE_KEY: 'EC private key',
    RSA_PRIVATE_KEY: 'RSA private key',
    DSA_PRIVATE_KEY: 'DSA private key',
    CERT_PRIVATE_KEY: 'Certificate private key',
    ENCRYPTED_PRIVATE_KEY: 'Encrypted private key',
    IMAGE_PULL_SECRET: 'Image pull secret',
});
