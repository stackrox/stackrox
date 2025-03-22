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
};

export type CosignPublicKey = {
    name: string;
    publicKeyPemEnc: string;
};

export type TransparencyLogVerification = {
    enabled: boolean;
    rekorUrl: string;
    ignoreSct: boolean;
    validateOffline: boolean;
};
