export type SignatureIntegration = {
    id: string;
    name: string;
    cosign: CosignPublicKeyVerification;
    cosignCertificates: CosignCertificateVerification[];
    transparencyLog: TransparencyLogVerification | null;
};

export type CosignPublicKeyVerification = {
    publicKeys: CosignPublicKey[];
};

export type CosignCertificateVerification = {
    certificateChainPemEnc: string;
    certificatePemEnc: string;
    certificateOidcIssuer: string;
    certificateIdentity: string;
    certificateTransparencyLog: CertificateTransparencyLogVerification | null;
    trustRoot: TrustRoot | null;
};

export type CertificateTransparencyLogVerification = {
    enabled: boolean;
    publicKeyPemEnc: string;
};

export type TrustRoot = {
    tufRepositoryUrl: string;
    tufRootUrl: string;
};

export type CosignPublicKey = {
    name: string;
    publicKeyPemEnc: string;
    trustRoot: TrustRoot | null;
};

export type TransparencyLogVerification = {
    enabled: boolean;
    publicKeyPemEnc: string;
    url: string;
    validateOffline: boolean;
};
