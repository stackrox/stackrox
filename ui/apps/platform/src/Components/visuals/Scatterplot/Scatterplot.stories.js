import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import { severities } from 'constants/severities';
import { severityColorMap } from 'constants/severityColors';
import Scatterplot from './Scatterplot';

export default {
    title: 'Scatterplot',
    component: Scatterplot,
};

const data = [
    { x: 6, y: 8.7, color: 'var(--caution-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 7, y: 4.9, color: 'var(--warning-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 43, y: 5.1, color: 'var(--warning-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 47, y: 2, color: 'var(--base-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 56, y: 8.2, color: 'var(--caution-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 59, y: 3.7, color: 'var(--base-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 65, y: 8.5, color: 'var(--caution-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 71, y: 6.6, color: 'var(--warning-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 80, y: 1.6, color: 'var(--base-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 81, y: 6.3, color: 'var(--warning-400)', url: '/main/configmanagement/cluster/88d17fde' },
    { x: 83, y: 9.1, color: 'var(--alert-400)', url: '/main/configmanagement/cluster/88d17fde' },
];
const legendData = [
    { title: 'Low', color: severityColorMap[severities.LOW_SEVERITY] },
    { title: 'Medium', color: severityColorMap[severities.MEDIUM_SEVERITY] },
    { title: 'High', color: severityColorMap[severities.HIGH_SEVERITY] },
    { title: 'Critical', color: severityColorMap[severities.CRITICAL_SEVERITY] },
];

export const withData = () => {
    return (
        <MemoryRouter>
            <div className="w-full h-64">
                <Scatterplot data={data} legendData={legendData} />
            </div>
        </MemoryRouter>
    );
};

export const withSetXDomain = () => {
    return (
        <MemoryRouter>
            <div className="w-full h-64">
                <Scatterplot data={data} lowerX={0} upperX={200} legendData={legendData} />
            </div>
        </MemoryRouter>
    );
};

export const withSetYDomain = () => {
    return (
        <MemoryRouter>
            <div className="w-full h-64">
                <Scatterplot data={data} lowerY={0} upperY={20} legendData={legendData} />
            </div>
        </MemoryRouter>
    );
};

export const withSetXandYDomains = () => {
    return (
        <MemoryRouter>
            <div className="w-full h-64">
                <Scatterplot
                    data={data}
                    lowerX={0}
                    upperX={150}
                    lowerY={0}
                    upperY={25}
                    legendData={legendData}
                />
            </div>
        </MemoryRouter>
    );
};

export const withExtraPaddingOnUpperBounds = () => {
    return (
        <MemoryRouter>
            <div className="w-full h-64">
                <Scatterplot
                    data={data}
                    lowerX={0}
                    lowerY={0}
                    xMultiple={5}
                    yMultiple={2}
                    shouldPadX
                    shouldPadY
                    legendData={legendData}
                />
            </div>
        </MemoryRouter>
    );
};
