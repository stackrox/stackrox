import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { podsData } from 'mockData/timelineData';
import Panel from 'Components/Panel';
import TimelineGraph from 'Components/TimelineGraph';
import EventTypeSelect from '../EventTypeSelect';
import getPodEvents from './getPodEvents';

// eslint-disable-next-line
const DeploymentEventTimeline = ({ id, goToNextView, selectedEventType, selectEventType }) => {
    // data fetching with "id", filtered by "selectedEventType" will happen here...
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

    const headerComponents = (
        <EventTypeSelect selectedEventType={selectedEventType} selectEventType={selectEventType} />
    );

    const timelineData = getPodEvents(podsData.pods, selectedEventType);

    return (
        <Panel header={header} headerComponents={headerComponents}>
            <TimelineGraph data={timelineData} goToNextView={goToNextView} />
        </Panel>
    );
};

DeploymentEventTimeline.propTypes = {
    id: PropTypes.string.isRequired,
    goToNextView: PropTypes.func.isRequired,
    selectedEventType: PropTypes.string.isRequired,
    selectEventType: PropTypes.func.isRequired
};

export default DeploymentEventTimeline;
