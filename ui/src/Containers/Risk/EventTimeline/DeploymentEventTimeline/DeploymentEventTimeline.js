import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { useQuery } from 'react-apollo';
import Raven from 'raven-js';

import Panel from 'Components/Panel';
import TimelineGraph from 'Components/TimelineGraph';
import Loader from 'Components/Loader';
import EventTypeSelect from '../EventTypeSelect';
import getPodEvents from './getPodEvents';
import { GET_DEPLOYMENT_EVENT_TIMELINE } from '../timelineQueries';

const DeploymentEventTimeline = ({ id, goToNextView, selectedEventType, selectEventType }) => {
    const { loading, error, data } = useQuery(GET_DEPLOYMENT_EVENT_TIMELINE, {
        variables: { deploymentId: id }
    });

    if (error) Raven.captureException(error);

    if (loading)
        return (
            <div className="py-4">
                <Loader message="Loading Event Timeline..." />
            </div>
        );

    const {
        numPolicyViolations,
        numProcessActivities,
        numRestarts,
        numTerminations,
        numTotalPods
    } = data.deployment;
    const numEvents = numPolicyViolations + numProcessActivities + numRestarts + numTerminations;

    const header = `${numEvents} ${pluralize(
        'event',
        numEvents
    )} across ${numTotalPods} ${pluralize('pod', numTotalPods)}`;

    const headerComponents = (
        <EventTypeSelect selectedEventType={selectedEventType} selectEventType={selectEventType} />
    );

    const timelineData = getPodEvents(data.pods, selectedEventType);

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
