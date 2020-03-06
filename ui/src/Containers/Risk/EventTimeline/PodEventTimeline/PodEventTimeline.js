import React from 'react';
import PropTypes from 'prop-types';
import { ArrowLeft } from 'react-feather';

import { getPodAndContainersByPodId } from 'mockData/timelineData';
import Button from 'Components/Button';
import Panel from 'Components/Panel';
import HeaderWithSubText from 'Components/HeaderWithSubText';
import TimelineGraph from 'Components/TimelineGraph';
import EventTypeSelect from '../EventTypeSelect';
import { getPod, getContainerEvents } from './getContainerEvents';

const PodEventTimeline = ({
    id,
    goToNextView,
    goToPreviousView,
    selectedEventType,
    selectEventType
}) => {
    const data = getPodAndContainersByPodId(id);
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

    return (
        <Panel headerTextComponent={headerTextComponent} headerComponents={headerComponents}>
            <TimelineGraph data={timelineData} goToNextView={goToNextView} />
        </Panel>
    );
};

PodEventTimeline.propTypes = {
    id: PropTypes.string.isRequired,
    goToNextView: PropTypes.func.isRequired,
    goToPreviousView: PropTypes.func.isRequired,
    selectedEventType: PropTypes.string.isRequired,
    selectEventType: PropTypes.func.isRequired
};

export default PodEventTimeline;
