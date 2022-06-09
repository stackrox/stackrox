import React from 'react';
import { Divider, Grid, GridItem, PageSection, Text, Title } from '@patternfly/react-core';
import SummaryCounts from './SummaryCounts';

import ViolationsByPolicyCategory from './Widgets/ViolationsByPolicyCategory';

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
            <PageSection>
                <Grid hasGutter>
                    <GridItem lg={6}>
                        <ViolationsByPolicyCategory />
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
}

export default DashboardPage;
