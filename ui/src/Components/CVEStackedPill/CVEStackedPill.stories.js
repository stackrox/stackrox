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

export const withURLAndHorizontalOrientation = () => (
    <MemoryRouter>
        <CVEStackedPill vulnCounter={vulnCounter} url="/main/" horizontal />
    </MemoryRouter>
);
