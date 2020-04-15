/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import TimelineGraph from './TimelineGraph';

export default {
    title: 'Timeline Graph',
    component: TimelineGraph
};

export const withData = () => {
    const [currentPage, onPageChange] = useState(1);
    const pageSize = 5;
    const data = [
        {
            type: 'graph-type-1',
            id: 'id-1',
            name: 'the podfather',
            subText: 'Started Jan 14, 1:00pm',
            events: [
                {
                    id: 'event-1',
                    differenceInHours: 5,
                    type: 'event-type-1',
                    edges: []
                },
                {
                    id: 'event-2',
                    differenceInHours: 2,
                    type: 'event-type-2',
                    edges: []
                },
                {
                    id: 'event-3',
                    differenceInHours: 1,
                    type: 'event-type-3',
                    edges: []
                }
            ]
        },
        {
            type: 'graph-type-1',
            id: 'id-2',
            name: 'poddy',
            subText: 'Started Jan 1, 1:00pm',
            events: [
                {
                    id: 'event-4',
                    differenceInHours: 5,
                    type: 'event-type-1',
                    edges: []
                },
                {
                    id: 'event-5',
                    differenceInHours: 6,
                    type: 'event-type-2',
                    edges: []
                },
                {
                    id: 'event-6',
                    differenceInHours: 9,
                    type: 'event-type-3',
                    edges: []
                }
            ]
        }
    ];
    return (
        <TimelineGraph
            data={data}
            currentPage={currentPage}
            totalSize={data.length}
            pageSize={pageSize}
            onPageChange={onPageChange}
            absoluteMaxTimeRange={10}
        />
    );
};
