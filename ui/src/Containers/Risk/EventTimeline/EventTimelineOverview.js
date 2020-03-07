import React, { useState } from 'react';
import PropTypes from 'prop-types';

import { graphObjectTypes } from 'constants/timelineTypes';
import { overviewData } from 'mockData/timelineData';
import Modal from 'Components/Modal';
import TimelineOverview from 'Components/TimelineOverview';
import EventTimeline from './EventTimeline';

const EventTimelineOverview = ({ deploymentId }) => {
    const [isModalOpen, setModalOpen] = useState(false);

    // data fetching with "deploymentId" will happen here...
    const {
        numPolicyViolations,
        numProcessActivities,
        numRestarts,
        numFailures
    } = overviewData.deployment;
    const numTotalEvents = numPolicyViolations + numProcessActivities + numRestarts + numFailures;

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
                <Modal
                    isOpen={isModalOpen}
                    onRequestClose={hideEventTimelineGraph}
                    className="w-2/3"
                >
                    <EventTimeline deploymentId={deploymentId} />
                </Modal>
            )}
        </>
    );
};

EventTimelineOverview.propTypes = {
    deploymentId: PropTypes.string.isRequired
};

export default React.memo(EventTimelineOverview);
