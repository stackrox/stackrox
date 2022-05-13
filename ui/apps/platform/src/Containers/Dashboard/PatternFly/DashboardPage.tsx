import React from 'react';
import { Divider, PageSection, Text, Title } from '@patternfly/react-core';

function DashboardPage() {
    return (
        <>
            <PageSection variant="light">
                <Title headingLevel="h1">Dashboard</Title>
                <Text>Review security metrics across all or select resources</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection />
        </>
    );
}

export default DashboardPage;
