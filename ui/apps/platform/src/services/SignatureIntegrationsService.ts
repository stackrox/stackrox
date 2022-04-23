import { SignatureIntegration } from 'types/signatureIntegration.proto';

import axios from './instance';

// eslint-disable-next-line import/prefer-default-export
export function fetchSignatureIntegrations(): Promise<SignatureIntegration[]> {
    return axios
        .get<{ integrations: SignatureIntegration[] }>('/v1/signatureintegrations')
        .then((response) => response?.data?.integrations ?? []);
}
