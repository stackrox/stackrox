import axios from 'services/instance';

type AIWorkloadType = 'UNKNOWN' | 'INFERENCE' | 'TRAINING';

export type AIWorkload = {
    id: string;
    namespace: string;
    name: string;
    clusterId: string;
    clusterName: string;
    workloadType: AIWorkloadType;
    modelFormat: string;
    storageUri: string;
    runtime: string;
    gpuRequests: string;
    cpuRequests: string;
    memoryRequests: string;
    deploymentMode: string;
    authEnabled: boolean;
    lastUpdated: string;
};

type ListAIWorkloadsResponse = {
    aiWorkloads: AIWorkload[];
    totalCount: number;
};

export function listAIWorkloads(): Promise<ListAIWorkloadsResponse> {
    return axios.get<ListAIWorkloadsResponse>('/v2/aiworkloads').then((response) => response.data);
}

export function getAIWorkload(id: string): Promise<AIWorkload> {
    return axios.get<AIWorkload>(`/v2/aiworkloads/${id}`).then((response) => response.data);
}
