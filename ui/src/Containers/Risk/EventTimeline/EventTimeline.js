import React, { useState } from 'react';
import PropTypes from 'prop-types';

import { eventTypes } from 'constants/timelineTypes';
import DeploymentEventTimeline from './DeploymentEventTimeline';

const EventTimeline = ({ deploymentId }) => {
    const [selectedEventType, selectEventType] = useState(eventTypes.ALL);
    // logic for determining what kind of event timeline goes here...
    // for now we'll default to the deployment event timeline
    return (
        <DeploymentEventTimeline
            deploymentId={deploymentId}
            selectedEventType={selectedEventType}
            selectEventType={selectEventType}
        />
    );
};

EventTimeline.propTypes = {
    deploymentId: PropTypes.string.isRequired
};

export default EventTimeline;
