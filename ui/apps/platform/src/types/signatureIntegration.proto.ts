export type SignatureIntegration = {
    id: string;
    name: string;
    cosign: CosignPublicKeyVerification;
    cosignCertificates: CosignCertificateVerification[];
    transparencyLog: TransparencyLogVerification;
};

export type CosignPublicKeyVerification = {
    publicKeys: CosignPublicKey[];
};

export type CosignCertificateVerification = {
    certificateChainPemEnc: string;
    certificatePemEnc: string;
    certificateOidcIssuer: string;
    certificateIdentity: string;
    certificateTransparencyLog: CertificateTransparencyLogVerification;
};

export type CertificateTransparencyLogVerification = {
    enabled: boolean;
    publicKeyPemEnc: string;
};

export type CosignPublicKey = {
    name: string;
    publicKeyPemEnc: string;
};

export type TransparencyLogVerification = {
    enabled: boolean;
    publicKeyPemEnc: string;
    url: string;
    validateOffline: boolean;
};
