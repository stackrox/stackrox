import React from 'react';
import { render } from '@testing-library/react';

import EventMarker from './EventMarker';

test('should show a policy violation event marker', async () => {
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                name="eventName"
                type="PolicyViolationEvent"
                differenceInHours={1}
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={2}
            />
        </svg>
    );
    expect(queryByTestId('policy-violation-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a process activity event marker', async () => {
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                name="eventName"
                type="ProcessActivityEvent"
                differenceInHours={1}
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={2}
            />
        </svg>
    );
    expect(queryByTestId('process-activity-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a container restart event marker', async () => {
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                name="eventName"
                type="ContainerRestartEvent"
                differenceInHours={1}
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={2}
            />
        </svg>
    );
    expect(queryByTestId('restart-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a container termination event marker', async () => {
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                name="eventName"
                type="ContainerTerminationEvent"
                differenceInHours={1}
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={2}
            />
        </svg>
    );
    expect(queryByTestId('termination-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});
