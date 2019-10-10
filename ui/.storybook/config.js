import React from 'react';
import { addDecorator, addParameters, configure } from '@storybook/react';
import { withInfo } from '@storybook/addon-info';

import { withThemes } from 'storybook-addon-themes';

addDecorator(
    withInfo({
        inline: true, // show headings, sourcecode, and props along with the examples,
        styles: stylesheet => ({
            ...stylesheet,
            infoBody: {
                ...stylesheet.infoBody,
                padding: '20px 0 40px'  // override left/right padding on info blocks to align with story wrapper padding below
            }
        })
    })
);

// whitespace around the story, which would otherwise start in the top-left
addDecorator(storyFn => <div style={{ padding: "10px 20px 20px" }}>{storyFn()}</div>);

addParameters({
    themes: [
        { name: 'Light Theme', class: 'theme-light', color: '#9199b1', default: true },
        { name: 'Dark Theme', class: 'theme-dark', color: '#5e667d' },
    ],
});

addDecorator(withThemes);

configure(require.context('../src', true, /\.stories\.js$/), module);
