import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import FixableCVECount from './FixableCVECount';

export default {
    title: 'FixableCVECount',
    component: FixableCVECount,
};

export const withCVE = () => <FixableCVECount cves={10} />;

export const withFixable = () => <FixableCVECount fixable={5} />;

export const withCVEAndFixable = () => <FixableCVECount cves={10} fixable={5} />;

export const withTotalURLOnly = () => (
    <MemoryRouter>
        <FixableCVECount cves={10} fixable={5} url="/to/some/where" />
    </MemoryRouter>
);

export const withVerticalOrientation = () => (
    <FixableCVECount cves={10} fixable={5} orientation="vertical" />
);

export const withUrlsAndVerticalOrientation = () => (
    <MemoryRouter>
        <FixableCVECount
            cves={10}
            fixable={5}
            url="/to/some/where"
            fixableUrl="/main?s[Is%20Fixable]=true"
            orientation="vertical"
        />
    </MemoryRouter>
);

export const withUrlsHiddenForPdf = () => (
    <MemoryRouter>
        <FixableCVECount
            cves={10}
            fixable={5}
            url="/to/some/where"
            fixableUrl="/main?s[Is%20Fixable]=true"
            orientation="vertical"
            hideLink
        />
    </MemoryRouter>
);
