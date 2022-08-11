import React from 'react';

import PluginProvider from 'console-plugins/PluginProvider';
import ViolationsByPolicyCategory from 'Containers/Dashboard/PatternFly/Widgets/ViolationsByPolicyCategory';

export default function Overview() {
    return (
        <PluginProvider>
            <ViolationsByPolicyCategory />
        </PluginProvider>
    );
}
