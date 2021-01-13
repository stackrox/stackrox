import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import SuccessButton from './SuccessButton';

export default {
    title: 'SuccessButton',
    component: SuccessButton,
} as Meta;

export const BasicUsage: Story = () => {
    function onClick(): void {}
    return (
        <SuccessButton type="button" onClick={onClick}>
            Save the Planet
        </SuccessButton>
    );
};
