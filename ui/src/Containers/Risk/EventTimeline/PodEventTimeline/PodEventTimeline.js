import React from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';
import { ArrowLeft } from 'react-feather';

import captureGraphQLErrors from 'modules/captureGraphQLErrors';
import queryService from 'modules/queryService';
import Button from 'Components/Button';
import Panel from 'Components/Panel';
import HeaderWithSubText from 'Components/HeaderWithSubText';
import TimelineGraph from 'Components/TimelineGraph';
import Loader from 'Components/Loader';
import EventTypeSelect from '../EventTypeSelect';
import { getPod, getContainerEvents } from './getContainerEvents';
import { GET_POD_EVENT_TIMELINE } from '../timelineQueries';

const defaultContainersSort = {
    id: 'Container Name',
    desc: false
};

const PodEventTimeline = ({
    id,
    goToNextView,
    goToPreviousView,
    selectedEventType,
    selectEventType,
    currentPage,
    pageSize,
    onPageChange
}) => {
    const { loading, error, data } = useQuery(GET_POD_EVENT_TIMELINE, {
        variables: {
            podId: id,
            // TODO: We should standardize on using Id vs. ID. Change this once backend makes the change
            containersQuery: queryService.objectToWhereClause({ 'Pod ID': id }),
            // TODO: Standardize on 1-indexing for Pagination so we can put the value adjustment into the function itself. https://github.com/stackrox/rox/pull/5075#discussion_r395284332
            pagination: queryService.getPagination(defaultContainersSort, currentPage - 1, pageSize)
        }
    });

    captureGraphQLErrors([error]);

    if (loading)
        return (
            <div className="flex flex-1 items-center justify-center py-4">
                <Loader message="Loading Event Timeline..." />
            </div>
        );

    const { name, subText } = getPod(data.pod);

    const headerTextComponent = (
        <>
            <Button
                dataTestId="timeline-back-button"
                className="border-base-300 border-r px-3 hover:bg-base-200"
                icon={<ArrowLeft className="h-4 w-4 text-base-600" />}
                onClick={goToPreviousView}
            />
            <HeaderWithSubText header={name} subText={subText} />
        </>
    );

    const headerComponents = (
        <EventTypeSelect selectedEventType={selectedEventType} selectEventType={selectEventType} />
    );

    const timelineData = getContainerEvents(data.containers, selectedEventType);

    const numTotalContainers = data?.pod?.liveInstances?.length || 0;

    return (
        <Panel headerTextComponent={headerTextComponent} headerComponents={headerComponents}>
            <TimelineGraph
                data={timelineData}
                goToNextView={goToNextView}
                currentPage={currentPage}
                totalSize={numTotalContainers}
                pageSize={pageSize}
                onPageChange={onPageChange}
            />
        </Panel>
    );
};

PodEventTimeline.propTypes = {
    id: PropTypes.string.isRequired,
    goToNextView: PropTypes.func.isRequired,
    goToPreviousView: PropTypes.func.isRequired,
    selectedEventType: PropTypes.string.isRequired,
    selectEventType: PropTypes.func.isRequired,
    currentPage: PropTypes.number.isRequired,
    pageSize: PropTypes.number.isRequired,
    onPageChange: PropTypes.func.isRequired,
    sort: PropTypes.shape({
        id: PropTypes.string.isRequired,
        desc: PropTypes.bool.isRequired
    }).isRequired
};

export default PodEventTimeline;
