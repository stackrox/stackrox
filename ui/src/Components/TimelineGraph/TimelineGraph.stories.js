import React from 'react';

import TimelineGraph from './TimelineGraph';

export default {
    title: 'Timeline Graph',
    component: TimelineGraph
};

export const withData = () => {
    const data = [
        {
            type: 'graph-type-1',
            id: 'id-1',
            name: 'name-1',
            subText: 'sub-text-1',
            events: [
                { id: 'event-1', timestamp: '2019-12-09T06:51:52Z', type: 'event-type-1' },
                { id: 'event-2', timestamp: '2019-12-09T06:51:52Z', type: 'event-type-2' },
                { id: 'event-3', timestamp: '2019-12-09T06:51:52Z', type: 'event-type-3' }
            ]
        },
        {
            type: 'graph-type-1',
            id: 'id-2',
            name: 'name-2',
            subText: 'sub-text-2',
            events: [
                { id: 'event-4', timestamp: '2019-12-09T06:51:52Z', type: 'event-type-1' },
                { id: 'event-5', timestamp: '2019-12-09T06:51:52Z', type: 'event-type-2' },
                { id: 'event-6', timestamp: '2019-12-09T06:51:52Z', type: 'event-type-3' }
            ]
        }
    ];
    return <TimelineGraph data={data} />;
};
