import { saveFile } from './DownloadService';

const certGenBaseURL = '/api/extensions/certgen';

export function generateCentralCertSecret() {
    return saveFile({
        method: 'post',
        url: `${certGenBaseURL}/central`,
    });
}

export function generateScannerCertSecret() {
    return saveFile({
        method: 'post',
        url: `${certGenBaseURL}/scanner`,
    });
}

export function generateSecuredClusterCertSecret(clusterId) {
    return saveFile({
        method: 'post',
        url: `${certGenBaseURL}/cluster`,
        data: { id: clusterId },
    });
}
