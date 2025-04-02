import FileSaver from 'file-saver';

import { GenerateClusterRegistrationSecretResponse } from 'services/ClustersService';

export function downloadClusterRegistrationSecret(
    name: string,
    response: GenerateClusterRegistrationSecretResponse
) {
    const { crs } = response;
    const decoded = typeof crs === 'string' ? window.atob(crs) : '';

    const file = new Blob([decoded], {
        type: 'application/x-yaml',
    });

    FileSaver.saveAs(file, `${name}-cluster-registration-secret.yaml`);
}
