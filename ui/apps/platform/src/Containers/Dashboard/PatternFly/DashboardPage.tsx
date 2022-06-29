import React from 'react';
import {
    Divider,
    Flex,
    FlexItem,
    Grid,
    GridItem,
    PageSection,
    Text,
    Title,
} from '@patternfly/react-core';
import SummaryCounts from './SummaryCounts';
import ScopeBar from './ScopeBar';

import ImagesAtMostRisk from './Widgets/ImagesAtMostRisk';
import ViolationsByPolicyCategory from './Widgets/ViolationsByPolicyCategory';
import DeploymentsAtMostRisk from './Widgets/DeploymentsAtMostRisk';
import AgingImages from './Widgets/AgingImages';
import ViolationsByPolicySeverity from './Widgets/ViolationsByPolicySeverity';

function DashboardPage() {
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <SummaryCounts />
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex
                    direction={{ default: 'column', lg: 'row' }}
                    alignItems={{ default: 'alignItemsFlexStart', lg: 'alignItemsCenter' }}
                >
                    <FlexItem>
                        <Title headingLevel="h1">Dashboard</Title>
                        <Text>Review security metrics across all or select resources</Text>
                    </FlexItem>
                    <FlexItem
                        grow={{ default: 'grow' }}
                        className="pf-u-display-flex pf-u-justify-content-flex-end"
                    >
                        <ScopeBar />
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <Grid hasGutter style={{ gridAutoRows: 'max-content' }}>
                    <GridItem lg={6}>
                        <ViolationsByPolicySeverity />
                    </GridItem>
                    <GridItem lg={6}>
                        <ImagesAtMostRisk />
                    </GridItem>
                    <GridItem lg={6}>
                        <DeploymentsAtMostRisk />
                    </GridItem>
                    <GridItem lg={6}>
                        <ViolationsByPolicyCategory />
                    </GridItem>
                    <GridItem lg={6}>
                        <AgingImages />
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
}

export default DashboardPage;
