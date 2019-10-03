import React from 'react';

import DashboardLayout from './index';

export default {
    title: 'DashboardLayout',
    component: DashboardLayout
};

export const withHeaderText = () => <DashboardLayout headerText="Storybook" />;

export const withHeaderComponents = () => (
    <DashboardLayout
        headerText="Storybook"
        headerComponents={<div>Header Components go here</div>}
    />
);

export const withChildren = () => (
    <DashboardLayout headerText="Storybook" headerComponents={<div>Header Components go here</div>}>
        <div className="text-base-800 flex items-center justify-center">
            Child Components go here
        </div>
    </DashboardLayout>
);
