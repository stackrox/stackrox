import React from 'react';
import { render, screen } from '@testing-library/react';

import { Event } from '../eventTypes';
import ClusteredEventMarker from './ClusteredEventMarker';

test('should show a clustered generic event marker', () => {
    const events: Event[] = Array.from(Array(9).keys()).map((index) => {
        return {
            id: `${index}`,
            name: `event-${index}`,
            type: 'PolicyViolationEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    events.push({
        id: '10',
        name: `event-10`,
        type: 'ProcessActivityEvent',
        args: '-g daemon off;',
        parentUid: -1,
        uid: 1000,
        timestamp: '2020-04-20T20:20:20.358227916Z',
    });
    const { asFragment } = render(
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
    expect(screen.getByTestId('clustered-generic-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a clustered policy violation event marker', () => {
    const events = Array.from(Array(10).keys()).map((index) => {
        return {
            id: index,
            name: `event-${index}`,
            type: 'PolicyViolationEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { asFragment } = render(
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
    expect(screen.getByTestId('clustered-policy-violation-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a clustered process activity event marker', () => {
    const events = Array.from(Array(10).keys()).map((index) => {
        return {
            id: index,
            name: `event-${index}`,
            type: 'ProcessActivityEvent',
            args: '-g daemon off;',
            parentName: null,
            parentUid: -1,
            uid: 1000,
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { asFragment } = render(
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
    expect(screen.getByTestId('clustered-process-activity-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a clustered process in baseline activity event marker', () => {
    const events = Array.from(Array(10).keys()).map((index) => {
        return {
            id: index,
            name: `event-${index}`,
            type: 'ProcessActivityEvent',
            args: '-g daemon off;',
            parentName: null,
            parentUid: -1,
            uid: 1000,
            inBaseline: true,
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { asFragment } = render(
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
    expect(screen.getByTestId('clustered-process-in-baseline-activity-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a clustered container restart event marker', () => {
    const events = Array.from(Array(10).keys()).map((index) => {
        return {
            id: index,
            name: `event-${index}`,
            type: 'ContainerRestartEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { asFragment } = render(
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
    expect(screen.getByTestId('clustered-restart-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});

test('should show a container termination event marker', () => {
    const events = Array.from(Array(10).keys()).map((index) => {
        return {
            id: index,
            name: `event-${index}`,
            type: 'ContainerTerminationEvent',
            reason: 'Because of Covid-19',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    const { asFragment } = render(
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
    expect(screen.getByTestId('clustered-termination-event')).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
});
