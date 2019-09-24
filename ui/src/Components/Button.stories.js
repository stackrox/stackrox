import React from 'react';
import { storiesOf } from '@storybook/react'; // eslint-disable-line import/no-extraneous-dependencies
import Button from './Button';

storiesOf('Button', module)
    .addParameters({
        themes: [
            { name: 'Light Theme', class: 'theme-light', color: '#9199b1', default: true },
            { name: 'Dark Theme', class: 'theme-dark', color: '#5e667d' }
        ]
    })
    .add('withText', () => <Button text="Make It So" />)
    .add('withTextCondensed', () => <Button text="Export" textCondensed="Export" />)
    .add('withClassName', () => <Button className="btn btn-base h-10" text="Export" />)
    .add('withTextClass', () => <Button text="Delete" textClass="text-alert-600" />)
    .add('withDisabled', () => <Button disabled text="Make It So" />)
    .add('withIsLoading', () => <Button isLoading text="Make It So" />);
