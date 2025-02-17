import React from 'react';
import { render, screen } from '@testing-library/react';

import EventMarker from './EventMarker';

test('should show a policy violation event marker', async () => {
    const { asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                id="1"
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
    expect(screen.getByTestId('policy-violation-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a process activity event marker', async () => {
    const { asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                id="1"
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
    expect(screen.getByTestId('process-activity-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a process in baseline activity event marker', async () => {
    const { asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                id="1"
                name="eventName"
                type="ProcessActivityEvent"
                timestamp="2020-04-20T20:20:20.358227916Z"
                args="-g daemon off;"
                parentName={null}
                parentUid={-1}
                uid={1000}
                differenceInMilliseconds={3600000}
                inBaseline
                translateX={0}
                translateY={0}
                size={10}
                minTimeRange={0}
                maxTimeRange={3600000 * 2}
            />
        </svg>
    );
    expect(screen.getByTestId('process-in-baseline-activity-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a container restart event marker', async () => {
    const { asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                id="1"
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
    expect(screen.getByTestId('restart-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a container termination event marker', async () => {
    const { asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <EventMarker
                id="1"
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
    expect(screen.getByTestId('termination-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});
