import React, { ReactElement } from 'react';
import { Story, StoryContext } from '@storybook/react/types-6-0';
import '@stackrox/tailwind-config/tailwind.css';
import '@stackrox/ui-components/lib/ui-components.css';

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
    return (
        <div className={context.globals.theme as string}>
            <div id="custom-root" className="flex font-sans h-full overflow-hidden text-base-600 text-base font-600 theme-light">
                {/* eslint-disable-next-line react/jsx-props-no-spreading */}
                <StoryComp {...context} />
            </div>
        </div>
    );
};

export const decorators = [withThemeProvider];
