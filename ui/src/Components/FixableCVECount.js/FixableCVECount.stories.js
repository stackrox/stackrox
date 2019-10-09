import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import FixableCVECount from './FixableCVECount';

export default {
    title: 'FixableCVECount',
    component: FixableCVECount
};

export const withCVE = () => <FixableCVECount cves={10} />;

export const withFixable = () => <FixableCVECount fixable={5} />;

export const withCVEAndFixable = () => <FixableCVECount cves={10} fixable={5} />;

export const withURL = () => (
    <MemoryRouter>
        <FixableCVECount cves={10} fixable={5} url="/to/some/where" />
    </MemoryRouter>
);

export const withVerticalOrientation = () => (
    <FixableCVECount cves={10} fixable={5} orientation="vertical" />
);

export const withURLAndVerticalOrientation = () => (
    <MemoryRouter>
        <FixableCVECount cves={10} fixable={5} url="/to/some/where" orientation="vertical" />
    </MemoryRouter>
);
