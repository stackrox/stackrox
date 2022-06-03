import React from 'react';
import { Divider, PageSection, Text, Title } from '@patternfly/react-core';
import SummaryCounts from './SummaryCounts';

function DashboardPage() {
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <SummaryCounts />
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Title headingLevel="h1">Dashboard</Title>
                <Text>Review security metrics across all or select resources</Text>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default DashboardPage;
