export type SignatureIntegration = {
    id: string;
    name: string;
    cosign: CosignPublicKeyVerification;
    cosignCertificates: CosignCertificateVerification[];
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
