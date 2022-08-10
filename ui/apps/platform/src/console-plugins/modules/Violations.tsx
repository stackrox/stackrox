import * as React from 'react';
import { PageSection } from '@patternfly/react-core';

import ViolationsPage from 'Containers/Violations/ViolationsPage';
import PluginProvider from 'console-plugins/PluginProvider';

export default function Violations() {
    return (
        <PluginProvider>
            <PageSection padding={{ default: 'noPadding' }} variant="default">
                <ViolationsPage />
            </PageSection>
        </PluginProvider>
    );
}
