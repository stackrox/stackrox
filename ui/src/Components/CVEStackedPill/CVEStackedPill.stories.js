import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import CVEStackedPill from './CVEStackedPill';

export default {
    title: 'CVEStackedPill',
    component: CVEStackedPill
};

const vulnCounter = {
    all: {
        total: 20,
        fixable: 4
    },
    critical: {
        total: 5,
        fixable: 1
    },
    high: {
        total: 5,
        fixable: 1
    },
    medium: {
        total: 5,
        fixable: 1
    },
    low: {
        total: 5,
        fixable: 1
    }
};

export const withVerticalOrientation = () => <CVEStackedPill vulnCounter={vulnCounter} />;

export const withHorizontalOrientation = () => (
    <CVEStackedPill vulnCounter={vulnCounter} horizontal />
);

export const withUrlsAndVerticalOrientation = () => (
    <MemoryRouter>
        <CVEStackedPill
            vulnCounter={vulnCounter}
            url="/main"
            fixableUrl="/main?s[Is%20Fixable]=true"
        />
    </MemoryRouter>
);

export const withUrlsAndHorizontalOrientation = () => (
    <MemoryRouter>
        <CVEStackedPill
            vulnCounter={vulnCounter}
            url="/main"
            fixableUrl="/main?s[Is%20Fixable]=true"
            horizontal
        />
    </MemoryRouter>
);

export const withUrlHiddenForPdf = () => (
    <MemoryRouter>
        <CVEStackedPill
            vulnCounter={vulnCounter}
            url="/main"
            fixableUrl="/main?s[Is%20Fixable]=true"
            horizontal
            hideLink
        />
    </MemoryRouter>
);
