import React, { useState } from 'react';
import PropTypes from 'prop-types';

import { eventTypes, rootTypes } from 'constants/timelineTypes';
import NotFoundMessage from 'Components/NotFoundMessage';
import DeploymentEventTimeline from './DeploymentEventTimeline';
import PodEventTimeline from './PodEventTimeline';

const EventTimelineComponentMap = {
    [rootTypes.DEPLOYMENT]: DeploymentEventTimeline,
    [rootTypes.POD]: PodEventTimeline
};

const EventTimeline = ({ deploymentId }) => {
    const rootView = [
        {
            type: rootTypes.DEPLOYMENT,
            id: deploymentId
        }
    ];

    const [selectedEventType, selectEventType] = useState(eventTypes.ALL);
    const [view, setView] = useState(rootView);

    function getCurrentView() {
        return view[view.length - 1];
    }

    function resetSelectedEventType() {
        selectEventType(eventTypes.ALL);
    }

    function goToRootView() {
        setView(rootView);
        resetSelectedEventType();
    }

    function goToNextView(type, id) {
        const newView = [...view, { type, id }];
        setView(newView);
        resetSelectedEventType();
    }

    function goToPreviousView() {
        if (view.length <= 1) return;
        setView(view.slice(0, -1));
        resetSelectedEventType();
    }

    const currentView = getCurrentView();

    const Component = EventTimelineComponentMap[currentView.type];
    if (!Component)
        return (
            <NotFoundMessage
                message="The Event Timeline for this view was not found."
                actionText="Go back"
                onClick={goToRootView}
            />
        );
    return (
        <Component
            id={currentView.id}
            goToNextView={goToNextView}
            goToPreviousView={goToPreviousView}
            selectedEventType={selectedEventType}
            selectEventType={selectEventType}
        />
    );
};

EventTimeline.propTypes = {
    deploymentId: PropTypes.string.isRequired
};

export default EventTimeline;
