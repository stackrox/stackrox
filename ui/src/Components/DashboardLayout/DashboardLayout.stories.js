import React from 'react';
import { storiesOf } from '@storybook/react'; // eslint-disable-line import/no-extraneous-dependencies
import DashboardLayout from './index';

storiesOf('Dashboard Layout', module)
    .addParameters({
        themes: [
            { name: 'Light Theme', class: 'theme-light', color: '#9199b1', default: true },
            { name: 'Dark Theme', class: 'theme-dark', color: '#5e667d' }
        ]
    })
    .add('withHeaderText', () => <DashboardLayout headerText="Storybook" />)
    .add('withHeaderComponents', () => (
        <DashboardLayout
            headerText="Storybook"
            headerComponents={<div>Header Components go here</div>}
        />
    ))
    .add('withChildren', () => (
        <DashboardLayout
            headerText="Storybook"
            headerComponents={<div>Header Components go here</div>}
        >
            <div className="text-base-800 flex items-center justify-center">
                Child Components go here
            </div>
        </DashboardLayout>
    ));
