import React from 'react';
import { Flex, FlexItem, Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import ComplianceByCluster from './Widgets/ComplianceByCluster';
import ComplianceByProfile from './Widgets/ComplianceByProfile';
import ScanResultsOverviewTable from './ScanResultsOverviewTable';

function ComplianceDashboardPage() {
    return (
        <>
            <PageTitle title="Compliance Status Dashboard" />
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
                    <Grid hasGutter md={6}>
                        <GridItem>
                            <ComplianceByCluster />
                        </GridItem>
                        <GridItem>
                            <ComplianceByProfile />
                        </GridItem>
                        <GridItem span={12}>
                            <ScanResultsOverviewTable />
                        </GridItem>
                    </Grid>
                </PageSection>
            </PageSection>
        </>
    );
}

export default ComplianceDashboardPage;
