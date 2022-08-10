import { PageSection, Title } from '@patternfly/react-core';
import PluginProvider from 'console-plugins/PluginProvider';
import ViolationsByPolicySeverity from 'Containers/Dashboard/PatternFly/Widgets/ViolationsByPolicySeverity';
import React from 'react';

export default function Overview() {
    return (
        <PluginProvider>
            <PageSection>
                <Title headingLevel="h2">System Overview</Title>
            </PageSection>
            <PageSection>
                <ViolationsByPolicySeverity />
            </PageSection>
        </PluginProvider>
    );
}
