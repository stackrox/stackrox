import React from 'react';

import BinderTabs from './BinderTabs';
import Tab from '../Tab';

export default {
    title: 'BinderTabs',
    component: BinderTabs,
};

export const withTabs = () => {
    return (
        <BinderTabs>
            <Tab title="tab 1">
                <div className="p-4">Tab 1 Content</div>
            </Tab>
            <Tab title="tab 2">
                <div className="p-4">Tab 2 Content</div>
            </Tab>
            <Tab title="tab 3">
                <div className="p-4">Tab 3 Content</div>
            </Tab>
        </BinderTabs>
    );
};
