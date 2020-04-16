import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { useQuery } from 'react-apollo';

import captureGraphQLErrors from 'modules/captureGraphQLErrors';
import queryService from 'modules/queryService';
import Panel from 'Components/Panel';
import TimelineGraph from 'Components/TimelineGraph';
import Loader from 'Components/Loader';
import EventTypeSelect from '../EventTypeSelect';
import getPodEvents from './getPodEvents';
import getLargestDifferenceInMilliseconds from '../eventTimelineUtils/getLargestDifferenceInMilliseconds';
import { GET_DEPLOYMENT_EVENT_TIMELINE } from '../timelineQueries';

const defaultPodsSort = {
    id: 'Pod Name',
    desc: false
};

const DeploymentEventTimeline = ({
    id,
    goToNextView,
    selectedEventType,
    selectEventType,
    currentPage,
    pageSize,
    onPageChange
}) => {
    const { loading, error, data } = useQuery(GET_DEPLOYMENT_EVENT_TIMELINE, {
        variables: {
            deploymentId: id,
            podsQuery: queryService.objectToWhereClause({ 'Deployment ID': id }),
            // TODO: Standardize on 1-indexing for Pagination so we can put the value adjustment into the function itself. https://github.com/stackrox/rox/pull/5075#discussion_r395284332
            pagination: queryService.getPagination(defaultPodsSort, currentPage - 1, pageSize)
        }
    });

    captureGraphQLErrors([error]);

    if (loading)
        return (
            <div className="flex flex-1 items-center justify-center py-4">
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
    const absoluteMaxTimeRange = getLargestDifferenceInMilliseconds(timelineData);

    return (
        <Panel header={header} headerComponents={headerComponents}>
            <TimelineGraph
                data={timelineData}
                goToNextView={goToNextView}
                currentPage={currentPage}
                totalSize={numTotalPods}
                pageSize={pageSize}
                onPageChange={onPageChange}
                absoluteMaxTimeRange={absoluteMaxTimeRange}
            />
        </Panel>
    );
};

DeploymentEventTimeline.propTypes = {
    id: PropTypes.string.isRequired,
    goToNextView: PropTypes.func.isRequired,
    selectedEventType: PropTypes.string.isRequired,
    selectEventType: PropTypes.func.isRequired,
    currentPage: PropTypes.number.isRequired,
    pageSize: PropTypes.number.isRequired,
    onPageChange: PropTypes.func.isRequired
};

export default DeploymentEventTimeline;
