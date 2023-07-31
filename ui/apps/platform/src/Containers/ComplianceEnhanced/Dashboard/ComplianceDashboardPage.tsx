import React from 'react';
import { Flex, FlexItem, Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import ComplianceByCluster from './Widgets/ComplianceByCluster';

function ComplianceDashboardPage() {
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
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Grid hasGutter md={6} xl2={4}>
                        <GridItem>
                            <ComplianceByCluster />
                        </GridItem>
                    </Grid>
                </PageSection>
            </PageSection>
        </>
    );
}

export default ComplianceDashboardPage;
