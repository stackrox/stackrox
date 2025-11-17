import type { ApiToken } from 'types/apiToken.proto';

import axios from './instance';

import type { Empty } from './types';

const url = '/v1/apitokens';

type GetAPITokensResponse = {
    tokens: ApiToken[];
};

/**
 * Fetches list of (unrevoked) API tokens.
 */
export function fetchAPITokens() {
    return axios.get<GetAPITokensResponse>(`${url}?revoked=false`).then((response) => ({
        response: response.data,
    }));
}

type ListAllowedTokenRolesResponse = {
    roleNames: string[];
};

export function fetchAllowedRoles() {
    return axios
        .get<ListAllowedTokenRolesResponse>(`${url}/generate/allowed-roles`)
        .then((response) => response.data.roleNames);
}

type GenerateTokenRequest = {
    name: string;
    roles: string[];
    expiration: string; // ISO 8601 data string
};

type GenerateTokenResponse = {
    token: string;
    metadata: ApiToken;
};

export function generateAPIToken(data: GenerateTokenRequest) {
    const options = {
        method: 'post',
        url: `${url}/generate`,
        data,
        // extend timeout to one minute, for https://stack-rox.atlassian.net/browse/ROX-5183
        timeout: 60000,
    };

    return axios<GenerateTokenResponse>(options).then((response) => ({
        response: response.data,
    }));
}

export function revokeAPIToken(id: string) {
    return axios.patch<Empty>(`${url}/revoke/${id}`).then((response) => response.data);
}

export function revokeAPITokens(ids: string[]) {
    return Promise.all(ids.map(revokeAPIToken));
}
