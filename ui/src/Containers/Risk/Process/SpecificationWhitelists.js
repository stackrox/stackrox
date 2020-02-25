import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import uniqBy from 'lodash/uniqBy';
import { getDeploymentAndProcessIdFromGroupedProcesses } from 'utils/processUtils';
import { fetchProcessesWhiteList } from 'services/ProcessesService';
import Whitelist from './Whitelist';

function loadWhitelists(deploymentId, processGroup, setProcessWhitelist) {
    const uniqueContainerNames = uniqBy(processGroup.groups, 'containerName').map(
        x => x.containerName
    );
    const { clusterId, namespace } = getDeploymentAndProcessIdFromGroupedProcesses(
        processGroup.groups
    );

    if (clusterId && namespace && uniqueContainerNames && uniqueContainerNames.length) {
        const promises = uniqueContainerNames.map(containerName => {
            const queryStr = `key.clusterId=${clusterId}&key.namespace=${namespace}&key.deploymentId=${deploymentId}&key.containerName=${containerName}`;
            return fetchProcessesWhiteList(queryStr);
        });
        Promise.all(promises).then(setProcessWhitelist);
    }
}

function SpecificationWhitelists({ deploymentId, processGroup, processEpoch, setProcessEpoch }) {
    const [processWhitelist, setProcessWhitelist] = useState(undefined);

    useEffect(
        () => {
            loadWhitelists(deploymentId, processGroup, setProcessWhitelist);
        },
        [deploymentId, processGroup, processEpoch]
    );

    if (!processWhitelist) return null;
    return (
        <div className="pl-3 pr-3">
            <ul className="list-reset border-b border-base-300 leading-normal hover:bg-primary-100">
                {processWhitelist.map(({ data }) => (
                    <Whitelist
                        process={data}
                        key={data.key.containerName}
                        processEpoch={processEpoch}
                        setProcessEpoch={setProcessEpoch}
                    />
                ))}
            </ul>
        </div>
    );
}

SpecificationWhitelists.propTypes = {
    deploymentId: PropTypes.string.isRequired,
    processGroup: PropTypes.shape({
        groups: PropTypes.arrayOf(PropTypes.object)
    }).isRequired,
    processEpoch: PropTypes.number.isRequired,
    setProcessEpoch: PropTypes.func.isRequired
};

export default SpecificationWhitelists;
