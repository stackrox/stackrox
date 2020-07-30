import React from 'react';

import ClusteredEventMarker from './ClusteredEventMarker';

export default {
    title: 'ClusteredEventMarker',
    component: ClusteredEventMarker,
};

export const clusteredEvent = () => {
    const events = [...Array(9).keys()].map((index) => {
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
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
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
};

export const clusteredPolicyViolationEvent = () => {
    const events = [...Array(10).keys()].map((index) => {
        return {
            name: `event-${index}`,
            type: 'PolicyViolationEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
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
};

export const clusteredProcessActivityEvent = () => {
    const events = [...Array(10).keys()].map((index) => {
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
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
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
};

export const clusteredWhitelistedProcessActivityEvent = () => {
    const events = [...Array(10).keys()].map((index) => {
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
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
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
};

export const clusteredRestartEvent = () => {
    const events = [...Array(10).keys()].map((index) => {
        return {
            name: `event-${index}`,
            type: 'ContainerRestartEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
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
};

export const clusteredTerminationEvent = () => {
    const events = [...Array(10).keys()].map((index) => {
        return {
            name: `event-${index}`,
            type: 'ContainerTerminationEvent',
            reason: 'Because of Covid-19',
            timestamp: '2020-04-20T20:20:20.358227916Z',
        };
    });
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
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
};
