import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { ClipLoader } from 'react-spinners';

const baseClass = 'py-6 border-b border-base-300 border-solid';

const YamlDownloadSection = ({ onFileDownload }) => (
    <div className="px-4">
        <div className={baseClass}>
            1) Download the required Configuration files
            <div className="flex justify-center p-3">
                <button
                    type="button"
                    className="download uppercase text-primary-600 p-2 text-center text-sm border border-solid bg-primary-200 border-primary-300 hover:bg-primary-100"
                    onClick={onFileDownload}
                    tabIndex="-1"
                >
                    <span className="pr-2">Download yaml file and keys</span>
                    <Icon.Download className="h-3 w-3" />
                </button>
            </div>
            <div className="text-xs text-center text-base-400">
                * You may modify the YAML to suit your environment as needed
            </div>
        </div>
        <div className={baseClass}>
            2) Use the deploy script inside the zip file to launch the sensor in your environment
        </div>
    </div>
);
YamlDownloadSection.propTypes = {
    onFileDownload: PropTypes.func.isRequired
};

const WaitingForCheckinMessage = () => (
    <div className="flex flex text-primary-600 bg-primary-200 border border-solid border-primary-400 p-4 items-center">
        <div className="text-center px-4">
            <ClipLoader color="currentColor" loading size={20} />
        </div>
        <div className="flex-3 pl-2">Waiting for the cluster to check-in successfully...</div>
    </div>
);

const SuccessfulCheckinMessage = () => (
    <div className="flex flex text-success-600 bg-success-200 border border-solid border-success-400 p-4 items-center">
        <div className="flex-1 text-center">
            <Icon.CheckCircle />
        </div>
        <div className="flex-3 pl-2">
            Success! The cluster has been recognized properly by StackRox. You may now save the
            configuration.
        </div>
    </div>
);

const ClusterCheckinSection = ({ clusterCheckedIn }) => (
    <div className="flex flex-col text-primary-500 p-4">
        {clusterCheckedIn ? <SuccessfulCheckinMessage /> : <WaitingForCheckinMessage />}
    </div>
);
ClusterCheckinSection.propTypes = {
    clusterCheckedIn: PropTypes.bool.isRequired
};

const ClusterDeploymentPage = ({ onFileDownload, clusterCheckedIn }) => (
    <div className="w-full">
        <YamlDownloadSection onFileDownload={onFileDownload} />
        <ClusterCheckinSection clusterCheckedIn={clusterCheckedIn} />
    </div>
);
ClusterDeploymentPage.propTypes = {
    onFileDownload: PropTypes.func.isRequired,
    clusterCheckedIn: PropTypes.bool.isRequired
};

export default ClusterDeploymentPage;
