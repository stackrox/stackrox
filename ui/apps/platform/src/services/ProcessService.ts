import type { ProcessIndicator } from 'types/processIndicator.proto';

import axios from './instance';

const baseUrl = '/v1/processes'; // RBAC resource: DeploymentExtension

export type ProcessGroup = {
    args: string;
    signals: ProcessIndicator[];
};

export type ProcessNameAndContainerNameGroup = {
    name: string;
    containerName: string;
    timesExecuted: number; // uint32
    groups: ProcessGroup[];
    suspicious: boolean;
};

type GetGroupedProcessesWithContainerResponse = {
    groups: ProcessNameAndContainerNameGroup[];
};

/**
 * Fetches Processes for a given deployment ID.
 * GetGroupedProcessByDeploymentAndContainer
 */
export function fetchProcesses(deploymentId: string): Promise<ProcessNameAndContainerNameGroup[]> {
    return axios
        .get<GetGroupedProcessesWithContainerResponse>(
            `${baseUrl}/deployment/${deploymentId}/grouped/container`
        )
        .then((response) => response?.data?.groups ?? []);
}
