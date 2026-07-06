export const secretTypes = [
    'UNDETERMINED',
    'PUBLIC_CERTIFICATE',
    'CERTIFICATE_REQUEST',
    'PRIVACY_ENHANCED_MESSAGE',
    'OPENSSH_PRIVATE_KEY',
    'PGP_PRIVATE_KEY',
    'EC_PRIVATE_KEY',
    'RSA_PRIVATE_KEY',
    'DSA_PRIVATE_KEY',
    'CERT_PRIVATE_KEY',
    'ENCRYPTED_PRIVATE_KEY',
    'IMAGE_PULL_SECRET',
] as const;

export type SecretType = (typeof secretTypes)[number];

export type ListSecret = {
    id: string;
    name: string;
    clusterId: string;
    clusterName: string;
    namespace: string;
    types: SecretType[];
    createdAt: string; // ISO 8601 string
};
