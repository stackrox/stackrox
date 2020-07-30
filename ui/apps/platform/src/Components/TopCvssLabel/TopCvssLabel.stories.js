import React from 'react';
import TopCvssLabel from './TopCvssLabel';

export default {
    title: 'TopCvssLabel',
    component: TopCvssLabel,
};

export const withData = () => (
    <div className="w-1/8">
        <TopCvssLabel cvss={3} version="V3" />
    </div>
);

export const expandedWithData = () => (
    <div className="w-1/8">
        <TopCvssLabel cvss={3} version="V3" expanded />
    </div>
);
