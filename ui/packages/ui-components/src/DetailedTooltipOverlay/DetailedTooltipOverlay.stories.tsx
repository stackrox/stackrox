import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import DetailedTooltipOverlay from './DetailedTooltipOverlay';

export default {
    title: 'DetailedTooltipOverlay',
    component: DetailedTooltipOverlay,
} as Meta;

const tooltipBody = (
    <ul className="flex-1 border-base-300 overflow-hidden">
        <li className="py-1 text-base-600 text-sm">Category: My Category</li>
        <li className="py-1 text-base-600 text-sm">Description: Self-described</li>
        <li className="py-1 text-base-600 text-sm">When It Happened: 11/19/2019 11:51:59AM</li>
    </ul>
);

export const TitleAndBody: Story<{}> = () => {
    return <DetailedTooltipOverlay title="scanner" body="Weighted CVSS: 6.7" />;
};

export const OptionalFooter: Story<{}> = () => {
    return (
        <DetailedTooltipOverlay
            title="scanner"
            body={tooltipBody}
            footer="I am at the very bottom..."
        />
    );
};

export const OptionalFooterAndSubtitle: Story<{}> = () => {
    return (
        <DetailedTooltipOverlay
            title="jon-snow"
            subtitle="remote/stackrox"
            body={tooltipBody}
            footer="I am at the very bottom..."
        />
    );
};
