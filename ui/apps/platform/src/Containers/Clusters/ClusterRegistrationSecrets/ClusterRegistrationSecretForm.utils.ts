import FileSaver from 'file-saver';

import { GenerateClusterRegistrationSecretResponse } from 'services/ClustersService';

export function downloadClusterRegistrationSecret(
    name: string,
    response: GenerateClusterRegistrationSecretResponse
) {
    const { crs } = response;
    // TODO atob is deprecated
    const decoded = typeof crs === 'string' ? atob(crs) : '';

    const file = new Blob([decoded], {
        type: 'application/x-yaml',
    });

    return FileSaver.saveAs(file, `${name}-cluster-registration-secret.yaml`);
}
