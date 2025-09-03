import React from 'react';
import { Flex, PageSection, Text, Title, Button } from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';

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
                    <Title headingLevel="h1">Virtual Machine Vulnerabilities</Title>
                    <Text>Prioritize and remediate observed CVEs across virtual machines</Text>
                    <div>
                        <Button variant="primary" onClick={handleViewDetails}>
                            Test VM Details
                        </Button>
                    </div>
                </Flex>
            </PageSection>
        </>
    );
}

export default VirtualMachineCvesOverviewPage;
