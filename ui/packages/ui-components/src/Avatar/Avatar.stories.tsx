import React from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';

import Avatar, { AvatarProps } from './Avatar';

export default {
    title: 'Avatar',
    component: Avatar,
    argTypes: {
        className: { table: { disable: true } },
        name: { control: 'text' },
        imageSrc: { control: 'text' },
    },
} as Meta;

const storiesExtraClassName = 'flex w-10 h-10 justify-center items-center';

export const Initials: Story<AvatarProps> = ({ name }) => (
    <Avatar name={name} extraClassName={storiesExtraClassName} />
);
Initials.args = {
    name: 'John Smith',
};
Initials.argTypes = {
    imageSrc: { table: { disable: true } },
};

export const NoName: Story<{}> = () => <Avatar extraClassName={storiesExtraClassName} />;
NoName.argTypes = {
    name: { table: { disable: true } },
    imageSrc: { table: { disable: true } },
};

export const Image: Story<AvatarProps> = ({ imageSrc }) => (
    <Avatar imageSrc={imageSrc} extraClassName={storiesExtraClassName} />
);
Image.args = {
    imageSrc: 'https://avatars1.githubusercontent.com/u/3277825',
};
Image.argTypes = {
    name: { table: { disable: true } },
};
