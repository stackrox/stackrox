import axios from './instance';
import { Empty } from './types';

const configTypes = ['GENERIC', 'GITHUB_ACTIONS'] as const;

export type MachineConfigType = (typeof configTypes)[number];

export type MachineConfigMapping = {
    key: string;
    valueExpression: string;
    role: string;
};

export type AuthMachineToMachineConfig = {
    id: string;
    tokenExpirationDuration: string;
    type: MachineConfigType;
    issuer: string;
    mappings: MachineConfigMapping[];
};

const machineAccessURL = `/v1/auth/m2m`;

export function fetchMachineAccessConfigs(): Promise<{
    response: { configs: AuthMachineToMachineConfig[] };
}> {
    return axios
        .get<{ configs: AuthMachineToMachineConfig[] }>(machineAccessURL)
        .then((response) => {
            return {
                response: response.data || { configs: [] },
            };
        });
}

export function deleteMachineAccessConfig(id: string): Promise<Empty> {
    return axios.delete(`${machineAccessURL}/${id}`);
}

export function deleteMachineAccessConfigs(ids: string[]): Promise<Empty[]> {
    return Promise.all(ids.map(deleteMachineAccessConfig));
}

export function createMachineAccessConfig(data: AuthMachineToMachineConfig): Promise<{
    response: AuthMachineToMachineConfig;
}> {
    return axios
        .post<AuthMachineToMachineConfig>(machineAccessURL, { config: data })
        .then((response) => {
            return {
                response: response.data || {},
            };
        });
}

export function updateMachineAccessConfig(data: AuthMachineToMachineConfig): Promise<Empty> {
    return axios.put(`${machineAccessURL}/${data.id}`, { config: data });
}
