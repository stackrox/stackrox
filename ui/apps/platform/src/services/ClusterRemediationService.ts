import axios from './instance';
import type { ImageRemediation } from './OperatorRemediationService';

export type JobStatus = 'STATUS_UNSPECIFIED' | 'RUNNING' | 'COMPLETE' | 'FAILED';

export type ClusterRemediationReport = {
    currentVersion: string;
    targetVersion: string;
    noUpdateReason: string;
    images: ImageRemediation[];
};

export type ClusterRemediationJob = {
    id: string;
    status: JobStatus;
    error: string;
    report?: ClusterRemediationReport;
};

const baseUrl = '/v1/cluster-remediation/reports';

// createClusterRemediationReport starts an OpenShift cluster-upgrade remediation report and returns
// a RUNNING job to poll.
export function createClusterRemediationReport(runningOnly: boolean): Promise<ClusterRemediationJob> {
    return axios
        .post<ClusterRemediationJob>(baseUrl, { runningOnly })
        .then((response) => response.data);
}

// getClusterRemediationReport polls a previously started report job.
export function getClusterRemediationReport(jobId: string): Promise<ClusterRemediationJob> {
    return axios
        .get<ClusterRemediationJob>(`${baseUrl}/${encodeURIComponent(jobId)}`)
        .then((response) => response.data);
}
