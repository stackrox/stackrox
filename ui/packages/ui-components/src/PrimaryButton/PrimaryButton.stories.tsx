import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import PrimaryButton from './PrimaryButton';

export default {
    title: 'PrimaryButton',
    component: PrimaryButton,
} as Meta;

export const BasicUsage: Story = () => {
    function onClick(): void {}
    return (
        <PrimaryButton type="button" onClick={onClick}>
            Save the Planet
        </PrimaryButton>
    );
};
