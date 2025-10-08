import React from 'react';
import { Alert, Card, CardBody, Flex, PageSection, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import DeveloperPreviewLabel from 'Components/PatternFly/DeveloperPreviewLabel';

import VirtualMachinesCvesTable from './VirtualMachinesCvesTable';

function VirtualMachineCvesOverviewPage() {
    return (
        <>
            <PageTitle title="Virtual Machine CVEs Overview" />
            <PageSection component="div" variant="light">
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <Title headingLevel="h1">Virtual Machine Vulnerabilities</Title>
                        <DeveloperPreviewLabel />
                    </Flex>
                    <Text>Prioritize and remediate observed CVEs across virtual machines</Text>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <Alert
                            className="pf-v5-u-m-md"
                            isInline
                            component="p"
                            variant="info"
                            title="The results only show vulnerabilities for DNF packages, that come from Red Hat repositories. The scan doesn't include System packages , which are preinstalled with the VM image and aren't tracked by the DNF database. Any DNF update could impact this default behavior."
                        />
                        <CardBody>
                            <VirtualMachinesCvesTable />
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
        </>
    );
}

export default VirtualMachineCvesOverviewPage;
