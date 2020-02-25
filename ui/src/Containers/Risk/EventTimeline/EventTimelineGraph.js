import React from 'react';
import PropTypes from 'prop-types';

const EventTimelineGraph = ({ deploymentId }) => {
    // data fetching with "deploymentId" will happen here...
    return <div className="p-4">Show Event Timeline Graph for Deployment: {deploymentId}</div>;
};

EventTimelineGraph.propTypes = {
    deploymentId: PropTypes.string.isRequired
};

export default EventTimelineGraph;
