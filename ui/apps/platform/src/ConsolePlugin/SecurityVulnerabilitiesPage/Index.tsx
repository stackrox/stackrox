import * as React from 'react';
import { PageSection, Title } from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';

import SummaryCounts from 'Containers/Dashboard/SummaryCounts';
import ViolationsByPolicyCategory from 'Containers/Dashboard/Widgets/ViolationsByPolicyCategory';
import PluginProvider from '../PluginProvider';

export function Index() {
    return (
        <PluginProvider>
            <PageSection>
                <Title headingLevel="h1">{'Hello, Plugin!'}</Title>
                <SummaryCounts
                    hasReadAccessForResource={{
                        Cluster: true,
                        Node: true,
                        Alert: true,
                        Deployment: true,
                        Image: true,
                        Secret: true,
                    }}
                />
                <ViolationsByPolicyCategory />
            </PageSection>
            <PageSection>
                <p>
                    <span className="console-plugin-template__nice">
                        <CheckCircleIcon /> {'Success!'}
                    </span>{' '}
                    {'Your plugin is working.'}
                </p>
            </PageSection>
        </PluginProvider>
    );
}
