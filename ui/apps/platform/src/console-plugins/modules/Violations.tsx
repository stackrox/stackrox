import * as React from 'react';

import ViolationsPage from 'Containers/Violations/ViolationsPage';
import PluginProvider from 'console-plugins/PluginProvider';

export default function Violations() {
    return (
        <PluginProvider>
            <ViolationsPage />
        </PluginProvider>
    );
}
