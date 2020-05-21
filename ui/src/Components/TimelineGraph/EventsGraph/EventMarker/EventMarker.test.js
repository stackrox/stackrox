import React from 'react';
import { render } from '@testing-library/react';

import EventMarker from './EventMarker';

test('should show a policy violation event marker', async () => {
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                name="eventName"
                type="PolicyViolationEvent"
                timestamp="2020-04-20T20:20:20.358227916Z"
                differenceInMilliseconds={3600000}
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={3600000 * 2}
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
                timestamp="2020-04-20T20:20:20.358227916Z"
                args="-g daemon off;"
                parentName={null}
                parentUid={-1}
                uid={1000}
                differenceInMilliseconds={3600000}
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={3600000 * 2}
            />
        </svg>
    );
    expect(queryByTestId('process-activity-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a whitelisted process activity event marker', async () => {
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                name="eventName"
                type="ProcessActivityEvent"
                timestamp="2020-04-20T20:20:20.358227916Z"
                args="-g daemon off;"
                parentName={null}
                parentUid={-1}
                uid={1000}
                differenceInMilliseconds={3600000}
                whitelisted
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={3600000 * 2}
            />
        </svg>
    );
    expect(queryByTestId('whitelisted-process-activity-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a container restart event marker', async () => {
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                name="eventName"
                type="ContainerRestartEvent"
                timestamp="2020-04-20T20:20:20.358227916Z"
                differenceInMilliseconds={3600000}
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={3600000 * 2}
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
                timestamp="2020-04-20T20:20:20.358227916Z"
                reason="OOMKilled"
                differenceInMilliseconds={3600000}
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={3600000 * 2}
            />
        </svg>
    );
    expect(queryByTestId('termination-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});
