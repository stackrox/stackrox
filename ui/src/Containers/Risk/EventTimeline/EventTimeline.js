import React, { useState } from 'react';
import PropTypes from 'prop-types';

import { graphObjectTypes } from 'constants/timelineTypes';
import Modal from 'Components/Modal';
import TimelineOverview from 'Components/TimelineOverview';
import EventTimelineGraph from './EventTimelineGraph';

const data = {
    deployment: {
        numPolicyViolations: 3,
        numProcessActivities: 5,
        numRestarts: 2,
        numFailures: 2
    }
};

const EventTimeline = ({ deploymentId }) => {
    const [isModalOpen, setModalOpen] = useState(false);

    // data fetching with "deploymentId" will happen here...
    const { numPolicyViolations, numProcessActivities, numRestarts, numFailures } = data.deployment;
    const numTotalEvents = Object.values(data.deployment).reduce((total, value) => total + value);

    function showEventTimelineGraph() {
        setModalOpen(true);
    }

    function hideEventTimelineGraph() {
        setModalOpen(false);
    }

    const counts = [
        { text: 'Policy Violations', count: numPolicyViolations },
        { text: 'Process Activities', count: numProcessActivities },
        { text: 'Restarts / Failures', count: numRestarts + numFailures }
    ];

    return (
        <>
            <TimelineOverview
                type={graphObjectTypes.EVENT}
                total={numTotalEvents}
                counts={counts}
                onClick={showEventTimelineGraph}
            />
            {isModalOpen && (
                <Modal isOpen={isModalOpen} onRequestClose={hideEventTimelineGraph}>
                    <EventTimelineGraph deploymentId={deploymentId} />
                </Modal>
            )}
        </>
    );
};

EventTimeline.propTypes = {
    deploymentId: PropTypes.string.isRequired
};

export default EventTimeline;
