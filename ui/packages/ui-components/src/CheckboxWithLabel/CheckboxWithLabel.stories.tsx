import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import CheckboxWithLabel from './CheckboxWithLabel';

export default {
    title: 'CheckboxWithLabel',
    component: CheckboxWithLabel,
} as Meta;

export const Checked: Story = () => {
    function onChange() {}
    return (
        <CheckboxWithLabel id="checked" ariaLabel="This is checked" checked onChange={onChange}>
            This is checked
        </CheckboxWithLabel>
    );
};

export const Unchecked: Story = () => {
    function onChange() {}
    return (
        <CheckboxWithLabel
            id="unchecked"
            ariaLabel="This is unchecked"
            checked={false}
            onChange={onChange}
        >
            This is unchecked
        </CheckboxWithLabel>
    );
};
