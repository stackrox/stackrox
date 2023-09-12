import React, { ReactElement } from 'react';
import { Alert, Button } from '@patternfly/react-core';
import { DownloadIcon } from '@patternfly/react-icons';
import { CheckCircle } from 'react-feather';
import { ClipLoader } from 'react-spinners';

import CollapsibleCard from 'Components/CollapsibleCard';
import ToggleSwitch from 'Components/ToggleSwitch';
import { ClusterManagerType } from 'types/cluster.proto';

const baseClass = 'py-6';

export type ClusterDeploymentProps = {
    clusterCheckedIn: boolean;
    createUpgraderSA: boolean;
    editing: boolean;
    isDownloadingBundle: boolean;
    managerType: ClusterManagerType;
    onFileDownload: () => void;
    toggleSA: () => void;
};

function ClusterDeployment({
    onFileDownload,
    isDownloadingBundle,
    clusterCheckedIn,
    editing,
    createUpgraderSA,
    toggleSA,
    managerType,
}: ClusterDeploymentProps): ReactElement {
    let managerTypeTitle = 'Dynamic configurations are automatically applied';
    let managerTypeText =
        'If you edited static configurations or you need to redeploy, download a new bundle.';
    if (managerType === 'MANAGER_TYPE_KUBERNETES_OPERATOR') {
        managerTypeTitle = 'Cluster labels have been saved';
        managerTypeText = 'All other cluster settings are managed by the Kubernetes operator.';
    }
    if (managerType === 'MANAGER_TYPE_HELM_CHART') {
        managerTypeTitle = 'Cluster labels have been saved';
        managerTypeText = 'All other cluster settings are managed by the Helm chart.';
    }
    return (
        <div className="md:max-w-sm">
            <div className="md:pr-4">
                {editing && clusterCheckedIn && (
                    <Alert variant="info" isInline title={managerTypeTitle} component="h3">
                        {managerTypeText}
                    </Alert>
                )}
                {managerType !== 'MANAGER_TYPE_KUBERNETES_OPERATOR' && (
                    <div className={baseClass}>
                        <CollapsibleCard title="1. Download files">
                            <div className="w-full h-full p-3 leading-normal">
                                <div className="border-b pb-3 mb-3 border-primary-300">
                                    Download the required configuration files, keys, and scripts.
                                </div>
                                <div className="flex items-center pb-2">
                                    <label
                                        htmlFor="createUpgraderSA"
                                        className="py-2 text-base-600 flex w-full"
                                    >
                                        Configure cluster to allow future automatic upgrades
                                    </label>
                                    <ToggleSwitch
                                        id="createUpgraderSA"
                                        toggleHandler={toggleSA}
                                        enabled={createUpgraderSA}
                                    />
                                </div>
                                <div className="flex justify-center px-3">
                                    <Button
                                        variant="secondary"
                                        icon={<DownloadIcon />}
                                        onClick={onFileDownload}
                                        isDisabled={isDownloadingBundle}
                                        isLoading={isDownloadingBundle}
                                    >
                                        Download YAML file and keys
                                    </Button>
                                </div>
                                <div className="py-2 text-xs text-center text-base-600">
                                    <p className="pb-2">
                                        Modify the YAML files to suit your environment if needed.
                                    </p>
                                    <p>Do not reuse this bundle for more than one cluster.</p>
                                </div>
                            </div>
                        </CollapsibleCard>
                        <div className="mt-4">
                            <CollapsibleCard title="2. Deploy">
                                <div className="w-full h-full p-3 leading-normal">
                                    Use the deploy script inside the bundle to set up your cluster.
                                </div>
                            </CollapsibleCard>
                        </div>
                    </div>
                )}
            </div>
            {(!editing || !clusterCheckedIn) && (
                <div className="flex flex-col text-primary-500 p-4">
                    {clusterCheckedIn ? (
                        <div className="flex text-success-600 bg-success-200 border border-solid border-success-400 p-4 items-center">
                            <div className="flex-1 text-center">
                                <CheckCircle />
                            </div>
                            <div className="flex-3 pl-2">
                                Success! The cluster has been recognized.
                            </div>
                        </div>
                    ) : (
                        <div className="flex text-primary-600 bg-primary-200 border border-solid border-primary-400 p-4 items-center">
                            <div className="text-center px-4">
                                <ClipLoader color="currentColor" loading size={20} />
                            </div>
                            <div className="flex-3 pl-2">
                                Waiting for the cluster to check in successfully...
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}

export default ClusterDeployment;
