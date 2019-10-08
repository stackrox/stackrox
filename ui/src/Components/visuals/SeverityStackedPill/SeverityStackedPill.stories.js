import React from 'react';

import SeverityStackedPill from './SeverityStackedPill';

export default {
    title: 'SeverityStackedPill',
    component: SeverityStackedPill
};

export const withData = () => {
    return <SeverityStackedPill low={25} medium={10} high={10} critical={5} />;
};

export const withPartialData = () => {
    return <SeverityStackedPill low={25} medium={10} />;
};
