import axios from './instance';
import { Empty } from './types';

export type MachineConfigType = 'GENERIC' | 'GITHUB_ACTIONS';

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

export function fetchMachineAccessConfigs(): Promise<{
    response: { configs: AuthMachineToMachineConfig[] };
}> {
    return axios.get<{ configs: AuthMachineToMachineConfig[] }>(`/v1/auth/m2m`).then((response) => {
        return {
            response: response.data || { configs: [] },
        };
    });
}

export function deleteMachineAccessConfig(id: string): Promise<Empty> {
    return axios.delete(`/v1/auth/m2m/${id}`);
}

export function deleteMachineAccessConfigs(ids: string[]): Promise<Empty[]> {
    return Promise.all(ids.map(deleteMachineAccessConfig));
}

export function createMachineAccessConfig(data: AuthMachineToMachineConfig): Promise<{
    response: AuthMachineToMachineConfig;
}> {
    return axios
        .post<AuthMachineToMachineConfig>(`/v1/auth/m2m`, { config: data })
        .then((response) => {
            return {
                response: response.data || {},
            };
        });
}
