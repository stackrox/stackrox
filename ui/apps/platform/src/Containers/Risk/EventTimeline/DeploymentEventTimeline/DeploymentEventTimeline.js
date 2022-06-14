import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { useQuery } from '@apollo/client';

import captureGraphQLErrors from 'utils/captureGraphQLErrors';
import queryService from 'utils/queryService';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import TimelineGraph from 'Components/TimelineGraph';
import Loader from 'Components/Loader';
import TimelineLegend from 'Components/TimelineLegend';
import ExportMenu from 'Containers/ExportMenu';
import EventTypeSelect from '../EventTypeSelect';
import getPodEvents from './getPodEvents';
import getLargestDifferenceInMilliseconds from '../eventTimelineUtils/getLargestDifferenceInMilliseconds';
import getTimelineQueryString from '../eventTimelineUtils/getTimelineQueryString';
import { GET_DEPLOYMENT_EVENT_TIMELINE } from '../timelineQueries';

const defaultPodsSort = {
    id: 'Pod Name',
    desc: false,
};

const DeploymentEventTimeline = ({
    id,
    goToNextView,
    selectedEventType,
    selectEventType,
    deploymentId,
    currentPage,
    pageSize,
    onPageChange,
    showClusteredEvents,
}) => {
    const { loading, error, data } = useQuery(GET_DEPLOYMENT_EVENT_TIMELINE, {
        variables: {
            deploymentId: id,
            podsQuery: queryService.objectToWhereClause({ 'Deployment ID': id }),
            // TODO: Standardize on 1-indexing for Pagination so we can put the value adjustment into the function itself. https://github.com/stackrox/stackrox/pull/5075#discussion_r395284332
            pagination: queryService.getPagination(defaultPodsSort, currentPage - 1, pageSize),
        },
    });

    captureGraphQLErrors([error]);

    if (loading) {
        return (
            <div className="flex flex-1 items-center justify-center py-4">
                <Loader message="Loading Event Timeline..." />
            </div>
        );
    }

    const {
        name,
        numPolicyViolations,
        numProcessActivities,
        numRestarts,
        numTerminations,
        numTotalPods,
    } = data.deployment;
    const numEvents = numPolicyViolations + numProcessActivities + numRestarts + numTerminations;

    const header = `${numEvents} ${pluralize(
        'event',
        numEvents
    )} across ${numTotalPods} ${pluralize('pod', numTotalPods)}`;

    const exportParams = {
        'Deployment ID': deploymentId,
    };
    const csvQueryString = getTimelineQueryString(exportParams);

    const timelineData = getPodEvents(data.pods, selectedEventType);
    const absoluteMaxTimeRange = getLargestDifferenceInMilliseconds(timelineData);

    return (
        <PanelNew testid="event-timeline">
            <PanelHead>
                <PanelTitle isUpperCase testid="event-timeline-header" text={header} />
                <PanelHeadEnd>
                    <EventTypeSelect
                        selectedEventType={selectedEventType}
                        selectEventType={selectEventType}
                    />
                    <div className="ml-3">
                        <TimelineLegend />
                    </div>
                    <div className="ml-3 mr-3">
                        <ExportMenu
                            fileName={`Event-Timeline-Report-${name}`}
                            pdfId="capture-timeline"
                            csvEndpoint="/api/risk/timeline/export/csv"
                            csvQueryString={csvQueryString}
                        />
                    </div>
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <TimelineGraph
                    key={selectedEventType}
                    data={timelineData}
                    goToNextView={goToNextView}
                    currentPage={currentPage}
                    totalSize={numTotalPods}
                    pageSize={pageSize}
                    onPageChange={onPageChange}
                    absoluteMaxTimeRange={absoluteMaxTimeRange}
                    showClusteredEvents={showClusteredEvents}
                />
            </PanelBody>
        </PanelNew>
    );
};

DeploymentEventTimeline.propTypes = {
    id: PropTypes.string.isRequired,
    goToNextView: PropTypes.func.isRequired,
    selectedEventType: PropTypes.string.isRequired,
    selectEventType: PropTypes.func.isRequired,
    deploymentId: PropTypes.string.isRequired,
    currentPage: PropTypes.number.isRequired,
    pageSize: PropTypes.number.isRequired,
    onPageChange: PropTypes.func.isRequired,
    showClusteredEvents: PropTypes.bool.isRequired,
};

export default DeploymentEventTimeline;
