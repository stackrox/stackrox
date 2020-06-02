import React from 'react';
import { render } from '@testing-library/react';

import ClusteredEventMarker from './ClusteredEventMarker';

test('should show a clustered generic event marker', async () => {
    const events = [...Array(9).keys()].map((_, index) => {
        return {
            name: `event-${index}`,
            type: 'PolicyViolationEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    events.push({
        name: `event-11`,
        type: 'ProcessActivityEvent',
        timestamp: '2020-04-20T20:20:20.358227916Z',
    });
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <ClusteredEventMarker
                events={events}
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={25}
            />
        </svg>
    );
    expect(queryByTestId('clustered-generic-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a clustered policy violation event marker', async () => {
    const events = [...Array(10).keys()].map((_, index) => {
        return {
            name: `event-${index}`,
            type: 'PolicyViolationEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <ClusteredEventMarker
                events={events}
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={25}
            />
        </svg>
    );
    expect(queryByTestId('clustered-policy-violation-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a clustered process activity event marker', async () => {
    const events = [...Array(10).keys()].map((_, index) => {
        return {
            name: `event-${index}`,
            type: 'ProcessActivityEvent',
            args: '-g daemon off;',
            parentName: null,
            parentUid: -1,
            uid: 1000,
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <ClusteredEventMarker
                events={events}
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={25}
            />
        </svg>
    );
    expect(queryByTestId('clustered-process-activity-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a clustered whitelisted process activity event marker', async () => {
    const events = [...Array(10).keys()].map((_, index) => {
        return {
            name: `event-${index}`,
            type: 'ProcessActivityEvent',
            args: '-g daemon off;',
            parentName: null,
            parentUid: -1,
            uid: 1000,
            whitelisted: true,
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <ClusteredEventMarker
                events={events}
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={25}
            />
        </svg>
    );
    expect(queryByTestId('clustered-whitelisted-process-activity-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a clustered container restart event marker', async () => {
    const events = [...Array(10).keys()].map((_, index) => {
        return {
            name: `event-${index}`,
            type: 'ContainerRestartEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <ClusteredEventMarker
                events={events}
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={25}
            />
        </svg>
    );
    expect(queryByTestId('clustered-restart-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a container termination event marker', async () => {
    const events = [...Array(10).keys()].map((_, index) => {
        return {
            name: `event-${index}`,
            type: 'ContainerTerminationEvent',
            reason: 'Because of Covid-19',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { queryByTestId, asFragment } = render(
        <svg height={100} width={100} data-testid="timeline-main-view">
            <ClusteredEventMarker
                events={events}
                type="ContainerTerminationEvent"
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={25}
            />
        </svg>
    );
    expect(queryByTestId('clustered-termination-event')).not.toBeNull();
    expect(asFragment()).toMatchSnapshot();
});
