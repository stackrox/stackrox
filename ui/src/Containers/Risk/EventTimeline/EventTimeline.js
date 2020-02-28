import React from 'react';
import PropTypes from 'prop-types';

import DeploymentEventTimeline from './DeploymentEventTimeline';

const EventTimeline = ({ deploymentId }) => {
    // logic for determining what kind of event timeline goes here...
    // for now we'll default to the deployment event timeline
    return <DeploymentEventTimeline deploymentId={deploymentId} />;
};

EventTimeline.propTypes = {
    deploymentId: PropTypes.string.isRequired
};

export default EventTimeline;
