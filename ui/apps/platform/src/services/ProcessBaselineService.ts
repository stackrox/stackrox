import type {
    ProcessBaseline,
    ProcessBaselineItem,
    ProcessBaselineKey,
} from 'types/processBaseline.proto';

import axios from './instance';

const baseUrl = '/v1/processbaselines'; // RBAC resource: DeploymentExtension

/**
 * Fetches container specific excluded scopes by deployment id and container id.
 * GetProcessBaseline
 */
export function fetchProcessesInBaseline(query: string): Promise<ProcessBaseline> {
    return axios.get<ProcessBaseline>(`${baseUrl}/key?${query}`).then((response) => response.data);
}

export type ProcessBaselineUpdateError = {
    error: string;
    key: ProcessBaselineKey;
};

export type UpdateProcessBaselinesResponse = {
    baselines: ProcessBaseline[];
    errors: ProcessBaselineUpdateError[];
};

export type LockProcessBaselinesRequest = {
    keys: ProcessBaselineKey[];
    locked: boolean;
};

/**
 * Lock/Unlock container specific process excluded scope by deployment id and container id.
 * LockProcessBaselines
 */
export function lockUnlockProcessBaselines(
    argument: LockProcessBaselinesRequest
): Promise<UpdateProcessBaselinesResponse> {
    return axios
        .put<UpdateProcessBaselinesResponse>(`${baseUrl}/lock`, argument)
        .then((response) => response.data);
}

export type AddProcessBaselinesRequest = {
    keys: ProcessBaselineKey[];
    addElements: ProcessBaselineItem[];
};

/**
 * Add container specific processes excluded scope by deployment id and container id.
 * UpdateProcessBaselines
 */
export function addProcessesToBaseline(
    argument: AddProcessBaselinesRequest
): Promise<UpdateProcessBaselinesResponse> {
    return axios
        .put<UpdateProcessBaselinesResponse>(`${baseUrl}`, argument)
        .then((response) => response.data);
}

export type RemoveProcessBaselinesRequest = {
    keys: ProcessBaselineKey[];
    removeElements: ProcessBaselineItem[];
};

/**
 * Remove container specific processes excluded scope by deployment id and container id.
 * UpdateProcessBaselines
 */
export function removeProcessesFromBaseline(
    argument: RemoveProcessBaselinesRequest
): Promise<UpdateProcessBaselinesResponse> {
    return axios
        .put<UpdateProcessBaselinesResponse>(`${baseUrl}`, argument)
        .then((response) => response.data);
}
