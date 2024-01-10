import FileSaver from 'file-saver';
import { FormikProps } from 'formik';

import { GenerateClusterInitBundleResponse } from 'services/ClustersService';

export const nameOfStep1 = 'Select options';
export const nameOfStep2 = 'Download bundle';

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

export type InitBundleWizardValues = {
    installation: InstallationKey;
    name: string;
    platform: PlatformKey;
};

export const initialValues: InitBundleWizardValues = {
    installation: 'Operator',
    name: '',
    platform: 'OpenShift',
};

export type InitBundleWizardFormikProps = FormikProps<InitBundleWizardValues>;

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
