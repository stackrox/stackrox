import React from 'react';
import { Flex, PageSection, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

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
        </>
    );
}

export default VirtualMachineCvesOverviewPage;
