import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import HelpIcon from './HelpIcon';

export default {
    title: 'HelpIcon',
    component: HelpIcon,
} as Meta;

export const BasicUsage: Story = () => {
    return <HelpIcon description="Remember to wash your hands thoroughly" />;
};
