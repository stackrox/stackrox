import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import uniqBy from 'lodash/uniqBy';
import { getDeploymentAndProcessIdFromGroupedProcesses } from 'utils/processUtils';
import { fetchProcessesInBaseline } from 'services/ProcessesService';
import ProcessBaselineList from './ProcessBaselineList';

function loadBaseline(deploymentId, processGroup, setProcessBaseline) {
    const uniqueContainerNames = uniqBy(processGroup.groups, 'containerName').map(
        (x) => x.containerName
    );
    const { clusterId, namespace } = getDeploymentAndProcessIdFromGroupedProcesses(
        processGroup.groups
    );

    if (clusterId && namespace && uniqueContainerNames && uniqueContainerNames.length) {
        const promises = uniqueContainerNames.map((containerName) => {
            const queryStr = `key.clusterId=${clusterId}&key.namespace=${namespace}&key.deploymentId=${deploymentId}&key.containerName=${containerName}`;
            return fetchProcessesInBaseline(queryStr);
        });
        Promise.all(promises).then(setProcessBaseline);
    }
}

function SpecificationBaselineList({ deploymentId, processGroup, processEpoch, setProcessEpoch }) {
    const [processBaseline, setProcessBaseline] = useState(undefined);

    useEffect(() => {
        loadBaseline(deploymentId, processGroup, setProcessBaseline);
    }, [deploymentId, processGroup, processEpoch]);

    if (!processBaseline) {
        return null;
    }
    return (
        <div className="pl-3 pr-3">
            <ul className="border-b border-base-300 leading-normal hover:bg-primary-100">
                {processBaseline.map(({ data }) => (
                    <ProcessBaselineList
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

SpecificationBaselineList.propTypes = {
    deploymentId: PropTypes.string.isRequired,
    processGroup: PropTypes.shape({
        groups: PropTypes.arrayOf(PropTypes.object),
    }).isRequired,
    processEpoch: PropTypes.number.isRequired,
    setProcessEpoch: PropTypes.func.isRequired,
};

export default SpecificationBaselineList;
