import React from 'react';
import { Flex, PageSection, Text, Title } from '@patternfly/react-core';

function CoveragesPageHeader() {
    return (
        <PageSection component="div" variant="light">
            <Flex direction={{ default: 'column' }}>
                <Title headingLevel="h1">Coverage</Title>
                <Text>
                    Assess profile compliance for nodes and platform resources across clusters
                </Text>
            </Flex>
        </PageSection>
    );
}

export default CoveragesPageHeader;
