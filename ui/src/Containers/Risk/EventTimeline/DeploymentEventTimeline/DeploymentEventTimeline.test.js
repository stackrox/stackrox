import React from 'react';
import { render } from '@testing-library/react';

import { eventTypes } from 'constants/timelineTypes';
import DeploymentEventTimeline from './DeploymentEventTimeline';

test('Should display total events across pods in a deployment', async () => {
    const id = '5039c79f-5294-11ea-84f7-025000000001';
    const selectedEventType = eventTypes.ALL;
    function selectEventType() {}
    function goToNextView() {}

    const { findByText } = render(
        <DeploymentEventTimeline
            id={id}
            goToNextView={goToNextView}
            selectedEventType={selectedEventType}
            selectEventType={selectEventType}
        />
    );

    findByText('12 EVENTS ACROSS 3 PODS');
});
