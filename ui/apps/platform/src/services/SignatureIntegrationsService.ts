import axios from './instance';

export type SignatureIntegration = {
    id: string;
    name: string;
    cosign: {
        publicKeys: {
            name: string;
            publicKeyPemEnc: string;
        }[];
    };
};

export function fetchSignatureIntegrations(): Promise<SignatureIntegration[]> {
    return axios
        .get<{ integrations: SignatureIntegration[] }>('/v1/signatureintegrations')
        .then((response) => response?.data?.integrations ?? []);
}
