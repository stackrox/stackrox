import React from 'react';
import {
    Card,
    CardBody,
    Flex,
    PageSection,
    Split,
    Text,
    Title,
    Button,
} from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';
import DeveloperPreviewLabel from 'Components/PatternFly/DeveloperPreviewLabel';

import VirtualMachinesCvesTable from './VirtualMachinesCvesTable';

function VirtualMachineCvesOverviewPage() {
    const navigate = useNavigate();

    const handleViewDetails = () => {
        navigate('test-vm-123');
    };

    return (
        <>
            <PageTitle title="Virtual Machine CVEs Overview" />
            <PageSection component="div" variant="light">
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Split hasGutter>
                        <Title headingLevel="h1">Virtual Machine Vulnerabilities</Title>
                        <DeveloperPreviewLabel />
                    </Split>
                    <Text>Prioritize and remediate observed CVEs across virtual machines</Text>
                    <div>
                        <Button variant="primary" onClick={handleViewDetails}>
                            Test VM Details
                        </Button>
                    </div>
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
