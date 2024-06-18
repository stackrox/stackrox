import React from 'react';
import { PageSection, Text, Title } from '@patternfly/react-core';

function CoveragesPageHeader() {
    return (
        <>
            <PageSection component="div" variant="light">
                <Title headingLevel="h1">Coverage</Title>
                <Text>
                    Assess profile compliance for nodes and platform resources across clusters
                </Text>
            </PageSection>
        </>
    );
}

export default CoveragesPageHeader;
