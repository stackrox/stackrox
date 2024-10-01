import axios from './instance';
import { Empty } from './types';

export type CloudSourceType = 'TYPE_UNSPECIFIED' | 'TYPE_PALADIN_CLOUD' | 'TYPE_OCM';

export type CloudSourceCredentials = {
    secret: string;
    clientId: string;
    clientSecret: string;
};

export type PaladinCloudConfig = {
    endpoint: string;
};
export type OcmConfig = {
    endpoint: string;
};

export type CloudSourceIntegration = {
    id: string;
    name: string;
    type: CloudSourceType;
    credentials: CloudSourceCredentials;
    skipTestIntegration: boolean;
    paladinCloud?: PaladinCloudConfig;
    ocm?: OcmConfig;
};

const cloudSourcesURL = `/v1/cloud-sources`;

export function fetchCloudSources(): Promise<{
    response: { cloudSources: CloudSourceIntegration[] };
}> {
    return axios.get(cloudSourcesURL).then((response) => ({
        response: response.data,
    }));
}

export function deleteCloudSource(id: string): Promise<Empty> {
    return axios.delete<Empty>(`${cloudSourcesURL}/${id}`).then((response) => response.data);
}

export function deleteCloudSources(ids: string[]): Promise<Empty[]> {
    return Promise.all(ids.map(deleteCloudSource));
}
