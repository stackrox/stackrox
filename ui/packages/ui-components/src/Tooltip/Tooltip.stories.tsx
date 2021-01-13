import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import * as Icon from 'react-feather';
import TooltipOverlay from '../TooltipOverlay';
import DetailedTooltipOverlay from '../DetailedTooltipOverlay';
import Tooltip from './Tooltip';

export default {
    title: 'Tooltip',
    component: Tooltip,
} as Meta;

export const ForButton: Story = () => (
    <Tooltip content={<TooltipOverlay>What does the octocat say?</TooltipOverlay>}>
        <button type="button">
            <Icon.GitHub />
            <span>Click me!</span>
        </button>
    </Tooltip>
);

export const ComplexTooltipContent: Story = () => (
    <Tooltip
        content={
            <DetailedTooltipOverlay
                title="Getting Nowhere"
                body="One can get quite far going nowhere..."
                footer="There is nothing philosophical in this message."
            />
        }
    >
        <a href="https://nowhere.com">Nowhere</a>
    </Tooltip>
);
