import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import LabeledBarGraph from './LabeledBarGraph';

export default {
    title: 'LabeledBarGraph',
    component: LabeledBarGraph
};

const data = [
    { x: 50, y: 'Process with UUID 0 / Enforced: Yes / Severity: High' },
    { x: 100, y: 'Latest Tag / Enforced: Yes / Severity: High' },
    { x: 25, y: 'Policy Name 1 / Enforced: Yes / Severity: Medium' },
    { x: 75, y: 'Policy Name 2 / Enforced: Yes / Severity: Medium' },
    { x: 30, y: 'Policy Name 3 / Enforced: Yes / Severity: Medium' },
    { x: 20, y: 'Policy Name 4 / Enforced: Yes / Severity: Low' },
    { x: 40, y: 'Policy Name 5 / Enforced: Yes / Severity: Low' }
];

export const withData = () => {
    return (
        <MemoryRouter>
            <LabeledBarGraph data={data} />
        </MemoryRouter>
    );
};

export const withTitle = () => {
    return (
        <MemoryRouter>
            <LabeledBarGraph data={data} title="Failing Deployments" />
        </MemoryRouter>
    );
};

export const withLinks = () => {
    const dataWithLinks = data.map(datum => {
        return {
            url: '/link/to/somewhere',
            ...datum
        };
    });
    return (
        <MemoryRouter>
            <LabeledBarGraph data={dataWithLinks} title="Failing Deployments" />
        </MemoryRouter>
    );
};

export const withFlexibleHeight = () => {
    const withHint = false;
    const largerDataSet = getLargerDataSet(withHint);

    return (
        <MemoryRouter>
            <LabeledBarGraph data={largerDataSet} title="Deployments" />
        </MemoryRouter>
    );
};

export const withTooltip = () => {
    const withHint = true;
    const largerDataSet = getLargerDataSet(withHint);

    return (
        <MemoryRouter>
            <div className="relative">
                <LabeledBarGraph data={largerDataSet} title="Deployments" />
            </div>
        </MemoryRouter>
    );
};

function getLargerDataSet(withHint) {
    const largerDataSet = [];
    for (let i = 0; i < 14; i += 1) {
        const randomX = Math.floor(Math.random() * 100);
        const name = `CVE-2019-${i} / CVSS 5.0 (v3.0)`;

        largerDataSet.push({
            x: randomX,
            y: name,
            hint: withHint
                ? {
                      title: `CVE-2019-${i}`,
                      body:
                          'Echo ethernet floating-point analog in computer plasma indeterminate integral interface inversion element.'
                  }
                : null
        });
    }

    return largerDataSet;
}
