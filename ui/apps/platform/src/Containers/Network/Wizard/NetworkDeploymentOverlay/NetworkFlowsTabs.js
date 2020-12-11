import React from 'react';

import BinderTabs from 'Components/BinderTabs';
import Tab from 'Components/Tab';

function NetworkFlowsTabs() {
    return (
        <BinderTabs>
            <Tab title="Active Flows">
                <div className="p-4">Active Flows</div>
            </Tab>
            <Tab title="Baseline Settings">
                <div className="p-4">Baseline Settings</div>
            </Tab>
        </BinderTabs>
    );
}

export default NetworkFlowsTabs;
