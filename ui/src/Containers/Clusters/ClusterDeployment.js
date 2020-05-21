import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { ClipLoader } from 'react-spinners';

import CollapsibleCard from 'Components/CollapsibleCard';
import Message from 'Components/Message';
import ToggleSwitch from 'Components/ToggleSwitch';

const baseClass = 'py-6';

const ClusterDeploymentPage = ({
    onFileDownload,
    clusterCheckedIn,
    editing,
    createUpgraderSA,
    toggleSA,
}) => (
    <div className="w-full">
        <div className="px-4">
            {editing && clusterCheckedIn && (
                <div className="w-full pb-3">
                    <Message
                        type="guidance"
                        message="Dynamic configurations are automatically applied.
                            If you edited static configurations or you need to redeploy, download a
                            new bundle."
                    />
                </div>
            )}
            <div className={baseClass}>
                <CollapsibleCard
                    title="1. Download files"
                    titleClassName="border-b px-1 border-primary-300 leading-normal cursor-pointer flex justify-between items-center bg-primary-200 hover:border-primary-400"
                >
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
                                name="createUpgraderSA"
                                toggleHandler={toggleSA}
                                enabled={createUpgraderSA}
                            />
                        </div>
                        <div className="flex justify-center px-3">
                            <button
                                type="button"
                                className="download uppercase text-primary-600 p-2 text-center text-sm border border-solid bg-primary-200 border-primary-300 hover:bg-primary-100"
                                onClick={onFileDownload}
                                tabIndex="-1"
                            >
                                <span className="pr-2">Download YAML file and keys</span>
                                <Icon.Download className="h-3 w-3" />
                            </button>
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
                    <CollapsibleCard
                        title="2. Deploy"
                        titleClassName="border-b px-1 border-primary-300 leading-normal cursor-pointer flex justify-between items-center bg-primary-200 hover:border-primary-400"
                    >
                        <div className="w-full h-full p-3 leading-normal">
                            Use the deploy script inside the bundle to set up your cluster.
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
        </div>
        {(!editing || !clusterCheckedIn) && (
            <div className="flex flex-col text-primary-500 p-4">
                {clusterCheckedIn ? (
                    <div className="flex flex text-success-600 bg-success-200 border border-solid border-success-400 p-4 items-center">
                        <div className="flex-1 text-center">
                            <Icon.CheckCircle />
                        </div>
                        <div className="flex-3 pl-2">
                            Success! The cluster has been recognized properly by StackRox.
                        </div>
                    </div>
                ) : (
                    <div className="flex flex text-primary-600 bg-primary-200 border border-solid border-primary-400 p-4 items-center">
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

ClusterDeploymentPage.propTypes = {
    onFileDownload: PropTypes.func.isRequired,
    clusterCheckedIn: PropTypes.bool.isRequired,
    editing: PropTypes.bool.isRequired,
    createUpgraderSA: PropTypes.bool.isRequired,
    toggleSA: PropTypes.func.isRequired,
};

export default ClusterDeploymentPage;
