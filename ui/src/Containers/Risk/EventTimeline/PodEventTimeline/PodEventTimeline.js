import React from 'react';
import PropTypes from 'prop-types';
import { ArrowLeft } from 'react-feather';

import Button from 'Components/Button';
import Panel from 'Components/Panel';

// eslint-disable-next-line
const PodEventTimeline = ({ id, goToPreviousView, selectedEventType, selectEventType }) => {
    // @TODO: Add stuff here in a future PR

    const headerTextComponent = (
        <>
            <Button
                dataTestId="timeline-back-button"
                className="border-base-300 border-r px-3 hover:bg-base-200"
                icon={<ArrowLeft className="h-4 w-4 text-base-600" />}
                onClick={goToPreviousView}
            />
            <div className="flex items-center font-700 text-base-600 px-3">
                Pod with Container Events
            </div>
        </>
    );

    return (
        <Panel headerTextComponent={headerTextComponent}>
            <div className="p-3">Events for Containers in this Pod show up here</div>
        </Panel>
    );
};

PodEventTimeline.propTypes = {
    id: PropTypes.string.isRequired,
    goToPreviousView: PropTypes.func.isRequired,
    selectedEventType: PropTypes.string.isRequired,
    selectEventType: PropTypes.func.isRequired
};

export default PodEventTimeline;
