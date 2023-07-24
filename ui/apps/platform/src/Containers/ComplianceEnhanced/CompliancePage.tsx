import React from 'react';
import { PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';

function CompliancePage() {
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-pl-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Compliance</Title>
                    </FlexItem>
                    <FlexItem>Benchmark compliance via profiles and clusters</FlexItem>
                </Flex>
            </PageSection>
        </>
    );
}

export default CompliancePage;
