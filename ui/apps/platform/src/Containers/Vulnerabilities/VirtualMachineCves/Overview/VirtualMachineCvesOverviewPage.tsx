import React from 'react';
import { Card, CardBody, Flex, PageSection, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

import VirtualMachinesCvesTable from './VirtualMachinesCvesTable';

function VirtualMachineCvesOverviewPage() {
    return (
        <>
            <PageTitle title="Virtual Machine CVEs Overview" />
            <PageSection component="div" variant="light">
                <Flex direction={{ default: 'column' }}>
                    <Title headingLevel="h1">Virtual Machine Vulnerabilities</Title>
                    <Text>Prioritize and remediate observed CVEs across virtual machines</Text>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
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
