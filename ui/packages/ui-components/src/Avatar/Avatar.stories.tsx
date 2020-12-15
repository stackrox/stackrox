import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import Avatar, { AvatarProps } from './Avatar';

export default {
    title: 'Avatar',
    component: Avatar,
    args: {
        extraClassName: 'flex w-10 h-10 justify-center items-center',
    },
    argTypes: {
        extraClassName: { table: { disable: true } },
        name: { control: 'text' },
        imageSrc: { control: 'text' },
    },
} as Meta;

export const Initials: Story<AvatarProps> = ({ name, extraClassName }) => (
    <Avatar name={name} extraClassName={extraClassName} />
);
Initials.args = {
    name: 'John Smith',
};
Initials.argTypes = {
    imageSrc: { table: { disable: true } },
};

export const NoName: Story<AvatarProps> = ({ extraClassName }) => (
    <Avatar extraClassName={extraClassName} />
);
NoName.argTypes = {
    name: { table: { disable: true } },
    imageSrc: { table: { disable: true } },
};

export const Image: Story<AvatarProps> = ({ imageSrc, extraClassName }) => (
    <Avatar imageSrc={imageSrc} extraClassName={extraClassName} />
);
Image.args = {
    imageSrc: 'https://avatars1.githubusercontent.com/u/3277825',
};
Image.argTypes = {
    name: { table: { disable: true } },
};
