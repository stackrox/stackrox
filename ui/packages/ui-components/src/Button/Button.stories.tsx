import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import Button from './Button';

export default {
    title: 'Button',
    component: Button,
} as Meta;

export const DefaultButton: Story<{}> = () => {
    function onClick(): void {}
    return (
        <Button type="button" onClick={onClick}>
            Click me
        </Button>
    );
};

export const SubmitButton: Story<{}> = () => {
    function onClick(): void {}
    return (
        <Button type="submit" onClick={onClick}>
            Click me
        </Button>
    );
};
