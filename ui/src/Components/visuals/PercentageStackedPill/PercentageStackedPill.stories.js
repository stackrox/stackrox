import React from 'react';

import PercentageStackedPill from './PercentageStackedPill';

export default {
    title: 'PercentageStackedPill',
    component: PercentageStackedPill
};

export const withData = () => {
    const data = [
        {
            colorType: 'base',
            value: 50
        },
        {
            colorType: 'warning',
            value: 20
        },
        {
            colorType: 'caution',
            value: 20
        },
        {
            colorType: 'alert',
            value: 10
        }
    ];
    return <PercentageStackedPill data={data} />;
};

export const withTooltip = () => {
    const data = [
        {
            colorType: 'base',
            value: 60
        },
        {
            colorType: 'warning',
            value: 40
        }
    ];
    const tooltip = {
        title: 'Criticality Distribution',
        body: (
            <div>
                <div>4 Medium CVES (2 Fixable)</div>
                <div>6 Low CVE</div>
            </div>
        )
    };
    return <PercentageStackedPill data={data} tooltip={tooltip} />;
};

export const withOneDataPoint = () => {
    const data = [
        {
            colorType: 'alert',
            value: 60
        }
    ];
    return <PercentageStackedPill data={data} />;
};

export const withTwoDataPoints = () => {
    const data = [
        {
            colorType: 'tertiary',
            value: 60
        },
        {
            colorType: 'alert',
            value: 40
        }
    ];
    return <PercentageStackedPill data={data} />;
};
