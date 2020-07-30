import React from 'react';

import * as Icon from 'react-feather';
import TooltipOverlay from 'Components/TooltipOverlay';
import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import Tooltip from './Tooltip';

export default {
    title: 'Tooltip',
    component: Tooltip,
};

export const withButton = () => (
    <Tooltip content={<TooltipOverlay>What does the octocat say?</TooltipOverlay>}>
        <button type="button">
            <Icon.GitHub />
            <span>Click me!</span>
        </button>
    </Tooltip>
);

export const withComplexTooltipContent = () => (
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
