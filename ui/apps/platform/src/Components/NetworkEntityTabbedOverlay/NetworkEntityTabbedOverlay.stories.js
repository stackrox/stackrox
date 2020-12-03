import React from 'react';

import Tab from 'Components/Tab';
import NetworkEntityTabbedOverlay from './NetworkEntityTabbedOverlay';

export default {
    title: 'NetworkEntityTabbedOverlay',
    component: NetworkEntityTabbedOverlay,
};

export const withOneTab = () => {
    return (
        <NetworkEntityTabbedOverlay>
            <Tab title="Tab 1">Tab 1 Content</Tab>
        </NetworkEntityTabbedOverlay>
    );
};

export const withTabs = () => {
    return (
        <NetworkEntityTabbedOverlay>
            <Tab title="Tab 1">Tab 1 Content</Tab>
            <Tab title="Tab 2">Tab 2 Content</Tab>
            <Tab title="Tab 3">Tab 3 Content</Tab>
        </NetworkEntityTabbedOverlay>
    );
};
