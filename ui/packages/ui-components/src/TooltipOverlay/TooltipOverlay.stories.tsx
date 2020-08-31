import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import TooltipOverlay from './TooltipOverlay';

export default {
    title: 'TooltipOverlay',
    component: TooltipOverlay,
} as Meta;

export const SimpleText: Story<{}> = () => (
    <TooltipOverlay extraClassName="w-20">Tooltip</TooltipOverlay>
);
