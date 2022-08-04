import { CertExpiryComponent } from 'types/credentialExpiryService.proto';

import { saveFile } from './DownloadService';

const certGenBaseURL = '/api/extensions/certgen';

const pathSegmentForComponent: Record<CertExpiryComponent, string> = {
    CENTRAL: 'central',
    SCANNER: 'scanner',
};

export function generateCertSecretForComponent(component: CertExpiryComponent) {
    return saveFile({
        method: 'post',
        url: `${certGenBaseURL}/${pathSegmentForComponent[component]}`,
        data: null,
    });
}

export function generateSecuredClusterCertSecret(clusterId) {
    return saveFile({
        method: 'post',
        url: `${certGenBaseURL}/cluster`,
        data: { id: clusterId },
    });
}
