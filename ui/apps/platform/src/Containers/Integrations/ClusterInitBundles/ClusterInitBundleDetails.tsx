import React, { ReactElement } from 'react';
import { Download } from 'react-feather';
import FileSaver from 'file-saver';

import LabeledValue from 'Components/LabeledValue';
import { ClusterInitBundle } from 'services/ClustersService';
import { getDateTime } from 'utils/dateUtils';

export type ClusterInitBundleDetailsProps = {
    authProviders: { name: string; id: string }[];
    clusterInitBundle: ClusterInitBundle;
    helmValuesBundle: Record<string, unknown> | null;
    kubectlBundle: Record<string, unknown> | null;
};

function ClusterInitBundleDetails({
    authProviders = [],
    clusterInitBundle,
    helmValuesBundle = null,
    kubectlBundle = null,
}: ClusterInitBundleDetailsProps): ReactElement | null {
    const authProviderName = authProviders.reduce((foundName, provider) => {
        if (clusterInitBundle.createdBy.authProviderId.includes(provider.id)) {
            return provider.name;
        }
        return foundName;
    }, '');
    const combinedProviderUserCombo = `${authProviderName}:${clusterInitBundle.createdBy.id}`;
    const decodedHelmValuesBundle =
        typeof helmValuesBundle === 'string' ? atob(helmValuesBundle) : '';
    const decodedKubectlBundle = typeof kubectlBundle === 'string' ? atob(kubectlBundle) : '';
    function onHelmValuesDownload() {
        const filename = `${clusterInitBundle.name}-cluster-init-bundle.yaml`;

        const file = new Blob([decodedHelmValuesBundle], {
            type: 'application/x-yaml',
        });

        FileSaver.saveAs(file, filename);
    }
    function onKubectlDownload() {
        const filename = `${clusterInitBundle.name}-cluster-init-secrets.yaml`;

        const file = new Blob([decodedKubectlBundle], {
            type: 'application/x-yaml',
        });

        FileSaver.saveAs(file, filename);
    }

    return (
        <div className="p-4 w-full" data-testid="bootstrap-token-details">
            <LabeledValue label="Name" value={clusterInitBundle.name} />
            <LabeledValue label="Issued" value={getDateTime(clusterInitBundle.createdAt)} />
            <LabeledValue label="Expiration" value={getDateTime(clusterInitBundle.expiresAt)} />
            <LabeledValue label="Created By" value={combinedProviderUserCombo} />

            {clusterInitBundle.createdBy.attributes.length > 0 && (
                <section className="border-t border-base-400 mt-2 pt-2">
                    <h2 className="font-700 mt-4 mb-2">Creator attributes:</h2>

                    {clusterInitBundle.createdBy.attributes.map((attribute) => (
                        <LabeledValue
                            key={attribute.key}
                            label={attribute.key}
                            value={attribute.value}
                        />
                    ))}
                </section>
            )}
            {decodedHelmValuesBundle.length > 0 && (
                <section className="border-t border-base-400 mt-2 py-2">
                    <p className="px-3 py-4">
                        Please copy the generated cluster init bundle YAML file and store it safely.
                        You will <strong>not</strong> be able to access it again after you close
                        this window.
                    </p>
                    <div className="flex justify-center px-6">
                        <button
                            type="button"
                            className="download uppercase text-primary-600 p-2 text-center text-sm border border-solid bg-primary-200 border-primary-300 hover:bg-primary-100 flex items-center"
                            onClick={onHelmValuesDownload}
                        >
                            <span className="pr-2">Download Helm values file</span>
                            <Download className="h-3 w-3" />
                        </button>
                    </div>
                    {decodedKubectlBundle.length > 0 && (
                        <>
                            <p className="px-3 py-4">
                                Use the following file if you do not want your secrets to be managed
                                by Helm. Most users should use the Helm values file above instead.
                            </p>
                            <div className="flex justify-center px-6">
                                <button
                                    type="button"
                                    className="download uppercase text-primary-600 p-2 text-center text-sm border border-solid bg-primary-200 border-primary-300 hover:bg-primary-100 flex items-center"
                                    onClick={onKubectlDownload}
                                >
                                    <span className="pr-2">Download Kubernetes secrets file</span>
                                    <Download className="h-3 w-3" />
                                </button>
                            </div>
                        </>
                    )}
                </section>
            )}
        </div>
    );
}

export default ClusterInitBundleDetails;
