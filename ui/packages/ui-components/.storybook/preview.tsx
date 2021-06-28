import React, { ReactElement } from 'react';
import { Story, StoryContext } from '@storybook/react/types-6-0';
import '@stackrox/tailwind-config/tailwind.css';

import '../lib/ui-components.css';

export const parameters = {
    actions: { argTypesRegex: '^on[A-Z].*' },
};

export const globalTypes = {
    theme: {
        name: 'Theme',
        description: 'Global theme for components',
        defaultValue: 'theme-light',
        toolbar: {
            icon: 'circlehollow',
            items: ['theme-light', 'theme-dark'],
        },
    },
};

const withThemeProvider = (StoryComp: Story, context: StoryContext): ReactElement => {
    const theme = context.globals.theme as string;
    const bgClass = theme === 'theme-dark' ? 'bg-base-0' : 'bg-base-100';
    return (
        <div
            className={`flex font-600 font-sans h-full overflow-hidden text-base text-base-600 ${theme} ${bgClass}`}
        >
            {/* eslint-disable-next-line react/jsx-props-no-spreading */}
            <StoryComp {...context} />
        </div>
    );
};

export const decorators = [withThemeProvider];
