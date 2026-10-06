import axios from './instance';

export type SeverityCounts = {
    critical: number;
    important: number;
    moderate: number;
    low: number;
    unknown: number;
};

export type ImageRemediation = {
    repository: string;
    name: string;
    installedImage: string;
    candidateImage: string;
    status: string;
    current: SeverityCounts;
    fixed: SeverityCounts;
    newCves: SeverityCounts;
};

export type ImageRemediationResponse = {
    operatorPackage: string;
    installedVersion: string;
    updateVersion: string;
    noUpdateReason: string;
    images: ImageRemediation[];
};

const baseUrl = '/v1/operator-remediation/image';

// getImageRemediation resolves the operator bundle shipping imageId and returns, per operator
// image, the current CVE counts by severity and the fixed/new changes from updating.
export function getImageRemediation(
    imageId: string,
    runningOnly: boolean
): Promise<ImageRemediationResponse> {
    return axios
        .get<ImageRemediationResponse>(`${baseUrl}/${encodeURIComponent(imageId)}`, {
            params: { runningOnly },
            // The backend scans candidate images live and can take ~2 min; override the 10s
            // instance default so the request is not aborted client-side.
            timeout: 300000,
        })
        .then((response) => response.data);
}
