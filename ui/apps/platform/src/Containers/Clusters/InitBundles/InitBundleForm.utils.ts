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
    response: GenerateClusterInitBundleResponse
) {
    const { helmValuesBundle, kubectlBundle } = response;
    const bundle = installation === 'Helm' ? helmValuesBundle : kubectlBundle;
    // TODO atob is deprecated
    const decoded = typeof bundle === 'string' ? atob(bundle) : '';

    const file = new Blob([decoded], {
        type: 'application/x-yaml',
    });
    const name =
        installation === 'Helm'
            ? 'Helm-values-cluster-init-bundle'
            : 'Operator-secrets-cluster-init-bundle';

    FileSaver.saveAs(file, `${name}.yaml`);
}
