import React from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import { PageSection, Title } from '@patternfly/react-core';

/**
 * Base Image detail page - placeholder for Phase 2
 */
function BaseImageDetailPage() {
    const { id } = useParams<{ id: string }>();

    return (
        <PageSection variant="light">
            <Title headingLevel="h1">Base Image Details</Title>
            <p>Base Image detail page for ID: {id}</p>
            <p>To be implemented in Phase 2</p>
        </PageSection>
    );
}

export default BaseImageDetailPage;
