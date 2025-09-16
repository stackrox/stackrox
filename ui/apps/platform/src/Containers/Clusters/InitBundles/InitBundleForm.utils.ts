import FileSaver from 'file-saver';

import { GenerateClusterInitBundleResponse } from 'services/ClustersService';

export const installationOptions: Record<string, string> = {
    Operator: 'Operator (recommended)',
    Helm: 'Helm chart',
} as const;

export type InstallationKey = keyof typeof installationOptions;

export const platformOptions: Record<string, string> = {
    OpenShift: 'OpenShift',
    EKS: 'EKS',
    AKS: 'AKS',
    GKE: 'GKE',
} as const;

export type PlatformKey = keyof typeof platformOptions;

export function downloadBundle(
    installation: InstallationKey,
    name: string,
    response: GenerateClusterInitBundleResponse
) {
    const { helmValuesBundle, kubectlBundle } = response;
    const bundle = installation === 'Helm' ? helmValuesBundle : kubectlBundle;
    const decoded = typeof bundle === 'string' ? window.atob(bundle) : '';

    const file = new Blob([decoded], {
        type: 'application/x-yaml',
    });
    const bundleName =
        installation === 'Helm'
            ? 'Helm-values-cluster-init-bundle'
            : 'Operator-secrets-cluster-init-bundle';

    FileSaver.saveAs(file, `${name}-${bundleName}.yaml`);
}
