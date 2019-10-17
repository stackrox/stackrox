import React from 'react';

import Scatterplot from './Scatterplot';

export default {
    title: 'Scatterplot',
    component: Scatterplot
};

const data = [
    { x: 6, y: 8.7, color: 'var(--caution-400)' },
    { x: 7, y: 4.9, color: 'var(--warning-400)' },
    { x: 43, y: 5.1, color: 'var(--warning-400)' },
    { x: 47, y: 2, color: 'var(--base-400)' },
    { x: 56, y: 8.2, color: 'var(--caution-400)' },
    { x: 59, y: 3.7, color: 'var(--base-400)' },
    { x: 65, y: 8.5, color: 'var(--caution-400)' },
    { x: 71, y: 6.6, color: 'var(--warning-400)' },
    { x: 80, y: 1.6, color: 'var(--base-400)' },
    { x: 81, y: 6.3, color: 'var(--warning-400)' },
    { x: 83, y: 9.1, color: 'var(--alert-400)' }
];

export const withData = () => {
    return <Scatterplot data={data} />;
};

export const withSetXDomain = () => {
    return <Scatterplot data={data} lowerX={0} upperX={200} />;
};

export const withSetYDomain = () => {
    return <Scatterplot data={data} lowerY={0} upperY={20} />;
};

export const withSetXandYDomains = () => {
    return <Scatterplot data={data} lowerX={0} upperX={150} lowerY={0} upperY={25} />;
};
