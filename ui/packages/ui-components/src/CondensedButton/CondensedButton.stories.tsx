import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import CondensedButton from './CondensedButton';

export default {
    title: 'CondensedButton',
    component: CondensedButton,
} as Meta;

export const DefaultButton: Story<{}> = () => {
    function onClick(): void {}
    return (
        <CondensedButton type="button" onClick={onClick}>
            Click me
        </CondensedButton>
    );
};
