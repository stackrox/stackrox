import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { useQuery } from '@apollo/client';
import Raven from 'raven-js';
import pluralize from 'pluralize';

import Modal from 'Components/Modal';
import TimelineOverview from 'Components/TimelineOverview';
import Loader from 'Components/Loader';
import EventTimeline from '../EventTimeline';
import { GET_EVENT_TIMELINE_OVERVIEW } from '../timelineQueries';

const EventTimelineOverview = ({ deploymentId }) => {
    const [isModalOpen, setModalOpen] = useState(false);
    const { loading, error, data } = useQuery(GET_EVENT_TIMELINE_OVERVIEW, {
        variables: { deploymentId },
    });

    if (error) {
        Raven.captureException(error);
    }

    if (loading) {
        return (
            <div className="bg-base-100 border border-primary-300 py-3">
                <Loader message="Loading Event Timeline..." />
            </div>
        );
    }

    // data fetching with "deploymentId" will happen here...
    const { numPolicyViolations, numProcessActivities, numRestarts, numTerminations } =
        data.deployment;
    const numRestartsAndTerminations = numRestarts + numTerminations;

    function showEventTimelineGraph() {
        setModalOpen(true);
    }

    function hideEventTimelineGraph() {
        setModalOpen(false);
    }

    const counts = [
        { text: pluralize('Policy Violation', numPolicyViolations), count: numPolicyViolations },
        {
            text: pluralize('Process Activities', numProcessActivities),
            count: numProcessActivities,
        },
        {
            text: pluralize('Restarts / Terminations', numRestartsAndTerminations),
            count: numRestartsAndTerminations,
        },
    ];

    return (
        <>
            <TimelineOverview
                dataTestId="event-timeline-overview"
                counts={counts}
                onClick={showEventTimelineGraph}
                loading={loading}
            />
            {isModalOpen && (
                <Modal isOpen={isModalOpen} onRequestClose={hideEventTimelineGraph}>
                    <EventTimeline deploymentId={deploymentId} />
                </Modal>
            )}
        </>
    );
};

EventTimelineOverview.propTypes = {
    deploymentId: PropTypes.string.isRequired,
};

export default React.memo(EventTimelineOverview);
