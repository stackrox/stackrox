import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import FormLabel from './FormLabel';

export default {
    title: 'FormLabel',
    component: FormLabel,
} as Meta;

export const Default: Story<{}> = () => (
    <FormLabel label="Name">
        <input type="text" className="form-input mt-3 bg-base-200" disabled />
    </FormLabel>
);

export const HelperText: Story<{}> = () => (
    <FormLabel label="Name" helperText="Write your name">
        <input type="text" className="form-input mt-3 bg-base-200" disabled />
    </FormLabel>
);

export const Required: Story<{}> = () => (
    <FormLabel label="Name" helperText="Write your name" isRequired>
        <input type="text" className="form-input mt-3 bg-base-200" disabled />
    </FormLabel>
);
