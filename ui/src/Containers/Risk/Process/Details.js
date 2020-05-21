import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';

import { fetchProcesses } from 'services/ProcessesService';
import { knownBackendFlags } from 'utils/featureFlags';
import FeatureEnabled from 'Containers/FeatureEnabled';
import EventTimelineOverview from '../EventTimeline/EventTimelineOverview';
import ProcessSpecificationWhitelists from './SpecificationWhitelists';
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
            <FeatureEnabled featureFlag={knownBackendFlags.ROX_EVENT_TIMELINE_UI}>
                <h3 className="border-b border-base-500 pb-2 mx-3 my-5">Event Timeline</h3>
                <div className="px-3">
                    <EventTimelineOverview deploymentId={deploymentId} />
                </div>
            </FeatureEnabled>
            <h3 className="border-b border-base-500 pb-2 mx-3 my-5">Running Processes</h3>
            <DiscoveryCards
                deploymentId={deploymentId}
                processGroup={processes}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
            <h3 className="border-b border-base-500 pb-2 mx-3 my-5">Spec Container Whitelists</h3>
            <ProcessSpecificationWhitelists
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
