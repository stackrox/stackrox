import React from 'react';

import HoverHintListItem from './HoverHintListItem';

export default {
    title: 'HoverHintListItem',
    component: HoverHintListItem,
};

export const withText = () => {
    return (
        <div className="bg-tertiary-200 relative min-h-8">
            <HoverHintListItem label="Weighted CVSS" value="8.9" />
        </div>
    );
};

export const withElements = () => {
    const label = (
        <span className="text-alert-700">
            <em>Danger!</em>
        </span>
    );
    const value = <span className="text-warning-800">Dr. Smith has the robot.</span>;

    return (
        <div className="bg-tertiary-200 relative min-h-8">
            <HoverHintListItem label={label} value={value} />
        </div>
    );
};
