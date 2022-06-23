import React, { useState } from 'react';
import {
    Divider,
    Flex,
    FlexItem,
    Grid,
    GridItem,
    PageSection,
    Split,
    SplitItem,
    Text,
    Title,
} from '@patternfly/react-core';
import { format } from 'date-fns';
import SummaryCounts from './SummaryCounts';
import ScopeBar from './ScopeBar';

import ViolationsByPolicyCategory from './Widgets/ViolationsByPolicyCategory';
import DeploymentsAtMostRisk from './Widgets/DeploymentsAtMostRisk';

function DashboardPage() {
    const [lastUpdate] = useState<Date>(new Date());
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Split className="pf-u-align-items-center">
                    <SplitItem isFilled>
                        <SummaryCounts />
                    </SplitItem>
                    <div
                        style={{ fontStyle: 'italic' }}
                        className="pf-u-color-200 pf-u-font-size-sm pf-u-mr-md pf-u-mr-lg-on-lg"
                    >
                        Last updated {format(lastUpdate, 'DD/MM/YYYY')} at{' '}
                        {format(lastUpdate, 'hh:mm A')}
                    </div>
                </Split>
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
                        <DeploymentsAtMostRisk />
                    </GridItem>
                    <GridItem lg={6}>
                        <ViolationsByPolicyCategory />
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
}

export default DashboardPage;
