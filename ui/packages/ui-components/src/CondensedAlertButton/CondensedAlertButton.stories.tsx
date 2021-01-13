import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import CondensedAlertButton from './CondensedAlertButton';

export default {
    title: 'CondensedAlertButton',
    component: CondensedAlertButton,
} as Meta;

export const DefaultButton: Story = () => {
    function onClick(): void {}
    return (
        <CondensedAlertButton type="button" onClick={onClick}>
            Click me
        </CondensedAlertButton>
    );
};
