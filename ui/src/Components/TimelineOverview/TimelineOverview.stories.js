import React from 'react';

import { graphObjectTypes } from 'constants/timelineTypes';
import TimelineOverview from './TimelineOverview';

export default {
    title: 'Timeline Overview',
    component: TimelineOverview
};

function onClick() {
    alert('You have triggered me!');
}

export const withNoCounts = () => {
    const counts = [];

    return <TimelineOverview type="EVENT" total={10} counts={counts} onClick={onClick} />;
};

export const withOneCount = () => {
    const counts = [{ text: 'Policy Violations', count: 5 }];

    return (
        <TimelineOverview
            type={graphObjectTypes.EVENT}
            total={10}
            counts={counts}
            onClick={onClick}
        />
    );
};

export const withMultipleCounts = () => {
    const counts = [
        { text: 'Policy Violations', count: 5 },
        { text: 'Process Activities', count: 10 },
        { text: 'Restarts / Failures', count: 15 }
    ];

    return <TimelineOverview type="EVENT" total={10} counts={counts} onClick={onClick} />;
};
