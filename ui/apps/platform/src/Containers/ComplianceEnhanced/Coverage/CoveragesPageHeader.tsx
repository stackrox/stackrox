import React from 'react';
import { PageSection, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

function CoveragesPageHeader() {
    return (
        <>
            <PageTitle title="Compliance coverage" />
            <PageSection component="div" variant="light">
                <Title headingLevel="h1">Compliance coverage</Title>
                <Text>
                    Assess profile compliance for nodes and platform resources across clusters
                </Text>
            </PageSection>
        </>
    );
}

export default CoveragesPageHeader;
