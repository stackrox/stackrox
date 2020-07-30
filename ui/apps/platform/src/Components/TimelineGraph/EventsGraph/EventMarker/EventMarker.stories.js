/* eslint-disable react-hooks/rules-of-hooks */
import React from 'react';

import EventMarker from './EventMarker';

export default {
    title: 'EventMarker',
    component: EventMarker,
};

export const policyViolationEvent = () => {
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
            <EventMarker
                name="event"
                type="PolicyViolationEvent"
                timestamp="2020-04-20T20:20:20.358227916Z"
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={20}
            />
        </svg>
    );
};

export const processActivityEvent = () => {
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
            <EventMarker
                name="event"
                type="ProcessActivityEvent"
                args="-g daemon off;"
                parentName={null}
                parentUid={-1}
                uid={1000}
                timestamp="2020-04-20T20:20:20.358227916Z"
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={20}
            />
        </svg>
    );
};

export const whitelistedProcessActivityEvent = () => {
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
            <EventMarker
                name="event"
                type="ProcessActivityEvent"
                args="-g daemon off;"
                parentName={null}
                parentUid={-1}
                uid={1000}
                whitelisted
                timestamp="2020-04-20T20:20:20.358227916Z"
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={20}
            />
        </svg>
    );
};

export const restartEvent = () => {
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
            <EventMarker
                name="event"
                type="ContainerRestartEvent"
                timestamp="2020-04-20T20:20:20.358227916Z"
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={20}
            />
        </svg>
    );
};

export const terminationEvent = () => {
    return (
        <svg data-testid="timeline-main-view" width={50} height={50}>
            <EventMarker
                name="event"
                type="ContainerTerminationEvent"
                reason="Because of Covid-19"
                timestamp="2020-04-20T20:20:20.358227916Z"
                differenceInMilliseconds={50}
                translateX={0}
                translateY={25}
                minTimeRange={0}
                maxTimeRange={100}
                size={20}
            />
        </svg>
    );
};
