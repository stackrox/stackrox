import React from 'react';
import { PageSection, Title, Divider, Flex, FlexItem, Button } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import usePermissions from 'hooks/usePermissions';

function VulnReportsPage() {
    const { hasReadWriteAccess, hasReadAccess } = usePermissions();

    const hasWorkflowAdministrationWriteAccess = hasReadWriteAccess('WorkflowAdministration');
    const hasImageReadAccess = hasReadAccess('Image');
    const hasAccessScopeReadAccess = hasReadAccess('Access');
    const hasNotifierIntegrationReadAccess = hasReadAccess('Integration');
    const canCreateReports =
        hasWorkflowAdministrationWriteAccess &&
        hasImageReadAccess &&
        hasAccessScopeReadAccess &&
        hasNotifierIntegrationReadAccess;

    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    className="pf-u-py-lg pf-u-px-lg"
                >
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Title headingLevel="h1">Vulnerability reporting</Title>
                            </FlexItem>
                            <FlexItem>
                                Configure reports, define report scopes, and assign distribution
                                lists to report on vulnerabilities across the organization.
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                    <FlexItem>
                        {canCreateReports && (
                            <Button variant="primary" onClick={() => {}}>
                                Create report
                            </Button>
                        )}
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }} />
        </>
    );
}

export default VulnReportsPage;
