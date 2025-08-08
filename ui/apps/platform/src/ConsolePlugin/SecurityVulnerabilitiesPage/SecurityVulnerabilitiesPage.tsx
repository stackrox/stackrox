import React from 'react';
import { PageSection, Title } from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';
import usePermissions from 'hooks/usePermissions';

import SummaryCounts from 'Containers/Dashboard/SummaryCounts';
import ViolationsByPolicyCategory from 'Containers/Dashboard/Widgets/ViolationsByPolicyCategory';

export function SecurityVulnerabilitiesPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAlert = hasReadAccess('Alert');
    const hasReadAccessForCluster = hasReadAccess('Cluster');
    const hasReadAccessForDeployment = hasReadAccess('Deployment');
    const hasReadAccessForImage = hasReadAccess('Image');
    const hasReadAccessForNode = hasReadAccess('Node');
    const hasReadAccessForSecret = hasReadAccess('Secret');

    return (
        <>
            <PageSection>
                <Title headingLevel="h1">{'Hello, Plugin!'}</Title>
                <SummaryCounts
                    hasReadAccessForResource={{
                        Cluster: hasReadAccessForCluster,
                        Node: hasReadAccessForNode,
                        Alert: hasReadAccessForAlert,
                        Deployment: hasReadAccessForDeployment,
                        Image: hasReadAccessForImage,
                        Secret: hasReadAccessForSecret,
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
        </>
    );
}
