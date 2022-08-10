import PluginProvider from 'console-plugins/PluginProvider';
import ViolationsByPolicySeverity from 'Containers/Dashboard/PatternFly/Widgets/ViolationsByPolicySeverity';
import React from 'react';

export default function Overview() {
    return (
        <PluginProvider>
            <ViolationsByPolicySeverity />
        </PluginProvider>
    );
}
