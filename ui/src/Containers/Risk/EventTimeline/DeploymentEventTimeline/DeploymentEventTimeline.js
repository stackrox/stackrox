import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { podsData } from 'mockData/timelineData';
import Panel from 'Components/Panel';
import TimelineGraph from 'Components/TimelineGraph';
import getPodEvents from './getPodEvents';

// eslint-disable-next-line
const DeploymentEventTimeline = ({ deploymentId }) => {
    // data fetching with "deploymentId", filtered by "selectedEventType" will happen here...
    const {
        numPolicyViolations,
        numProcessActivities,
        numRestarts,
        numFailures,
        numTotalPods
    } = podsData.deployment;
    const numEvents = numPolicyViolations + numProcessActivities + numRestarts + numFailures;

    const header = `${numEvents} ${pluralize(
        'event',
        numEvents
    )} across ${numTotalPods} ${pluralize('pod', numTotalPods)}`;

    const headerComponents = null;

    const timelineData = getPodEvents(podsData.pods);

    return (
        <Panel header={header} headerComponents={headerComponents}>
            <TimelineGraph data={timelineData} />
        </Panel>
    );
};

DeploymentEventTimeline.propTypes = {
    deploymentId: PropTypes.string.isRequired
};

export default DeploymentEventTimeline;
