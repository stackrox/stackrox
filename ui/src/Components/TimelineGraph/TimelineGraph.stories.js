/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import TimelineGraph from './TimelineGraph';

export default {
    title: 'Timeline Graph',
    component: TimelineGraph,
};

export const basicUsage = () => {
    const [currentPage, onPageChange] = useState(1);
    const pageSize = 5;
    const data = [
        {
            type: 'POD',
            id: 'f3f5dbfb-3133-505f-860b-fec763d5665c',
            name: 'monitoring-74f68cdfbc-q4qm9',
            subText: 'Started Jun 01, 10:06AM',
            drillDownButtonTooltip: 'View 3 Containers',
            hasChildren: true,
            events: [
                {
                    id: `event-1`,
                    name: `event-1`,
                    type: 'PolicyViolationEvent',
                    timestamp: '2020-04-20T20:21:20.358227916Z',
                    differenceInMilliseconds: 10000,
                },
                {
                    id: `event-2`,
                    name: `event-2`,
                    type: 'PolicyViolationEvent',
                    timestamp: '2020-04-20T20:22:20.358227916Z',
                    differenceInMilliseconds: 11785,
                },
            ],
        },
    ];
    return (
        <TimelineGraph
            data={data}
            currentPage={currentPage}
            totalSize={data.length}
            pageSize={pageSize}
            onPageChange={onPageChange}
            absoluteMaxTimeRange={60000}
        />
    );
};

export const withClusteredEvents = () => {
    const [currentPage, onPageChange] = useState(1);
    const pageSize = 5;

    const clusteredEvents = [...Array(10).keys()].map((index) => {
        return {
            id: `event-${index}`,
            name: `event-${index}`,
            type: 'PolicyViolationEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
            differenceInMilliseconds: 2500,
        };
    });
    clusteredEvents.push({
        id: `event-10`,
        name: `event-10`,
        type: 'PolicyViolationEvent',
        timestamp: '2020-04-20T20:20:20.358227916Z',
        differenceInMilliseconds: 2499,
    });
    clusteredEvents.push({
        id: `event-11`,
        name: `event-11`,
        type: 'PolicyViolationEvent',
        timestamp: '2020-04-20T20:20:20.358227916Z',
        differenceInMilliseconds: 3215,
    });

    const multiClusteredEvents = [...Array(4).keys()].map((index) => {
        return {
            id: `event-${index}`,
            name: `event-${index}`,
            type: 'PolicyViolationEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
            differenceInMilliseconds: 5000,
        };
    });
    multiClusteredEvents.push({
        name: `event-5`,
        type: 'ProcessActivityEvent',
        timestamp: '2020-04-20T20:20:20.358227916Z',
        differenceInMilliseconds: 5000,
    });

    const data = [
        {
            type: 'POD',
            id: 'f3f5dbfb-3133-505f-860b-fec763d5665c',
            name: 'monitoring-74f68cdfbc-q4qm9',
            subText: 'Started Jun 01, 10:06AM',
            drillDownButtonTooltip: 'View 3 Containers',
            hasChildren: true,
            events: clusteredEvents,
        },
        {
            type: 'POD',
            id: 'g3f5dbfb-3133-505f-860b-fec763d5665c',
            name: 'nginx-74f68cdfbc-q4qm9',
            subText: 'Started Jun 01, 10:06AM',
            drillDownButtonTooltip: 'View 3 Containers',
            hasChildren: true,
            events: multiClusteredEvents,
        },
    ];
    return (
        <TimelineGraph
            data={data}
            currentPage={currentPage}
            totalSize={data.length}
            pageSize={pageSize}
            onPageChange={onPageChange}
            absoluteMaxTimeRange={10000}
        />
    );
};
