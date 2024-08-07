import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';

import { fetchProcesses } from 'services/ProcessesService';
import EventTimelineOverview from '../EventTimeline/EventTimelineOverview';
import SpecificationBaselineList from './SpecificationBaselineList';
import DiscoveryCards from './DiscoveryCards';

function Details({ deploymentId, processGroup }) {
    const [processEpoch, setProcessEpoch] = useState(0);
    const [processes, setProcesses] = useState(processGroup);

    useEffect(() => {
        if (processEpoch === 0) {
            return;
        }
        fetchProcesses(deploymentId).then((resp) => setProcesses(resp.response));
    }, [deploymentId, setProcesses, processEpoch]);

    return (
        <div>
            <h3 className="border-b border-base-500 pb-2 mx-3 my-5">Event Timeline</h3>
            <div className="px-3">
                <EventTimelineOverview deploymentId={deploymentId} />
            </div>
            <h3 className="border-b border-base-500 pb-2 mx-3 my-5">
                History of Running Processes
            </h3>
            <DiscoveryCards
                deploymentId={deploymentId}
                processGroup={processes}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
            <h3 className="border-b border-base-500 pb-2 mx-3 my-5">Spec Container Baselines</h3>
            <SpecificationBaselineList
                deploymentId={deploymentId}
                processGroup={processes}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
        </div>
    );
}

Details.propTypes = {
    deploymentId: PropTypes.string.isRequired,
    processGroup: PropTypes.shape({
        groups: PropTypes.arrayOf(PropTypes.object),
    }).isRequired,
};

export default Details;
