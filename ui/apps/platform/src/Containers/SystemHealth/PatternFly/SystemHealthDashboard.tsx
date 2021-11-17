import React, { ReactElement, useState } from 'react';
import {
    Card,
    CardHeader,
    CardHeaderMain,
    CardActions,
    CardTitle,
    CardBody,
    Flex,
    FlexItem,
    Grid,
    GridItem,
    PageSection,
    PageSectionVariants,
    Title,
} from '@patternfly/react-core';

import ViewAllButton from 'Components/PatternFly/ViewAllButton';
import useInterval from 'hooks/useInterval';
import { clustersBasePath } from 'routePaths';
import VulnerabilityDefinitionsWidget from './Components/VulnerabilityDefinitionsWidget';
import GenerateDiagnosticBundle from './Components/GenerateDiagnosticBundle';
import ImageIntegrationHealthWidget from './Components/ImageIntegrationHealthWidget';
import NotifierIntegrationHealthWidget from './Components/NotifierIntegrationHealthWidget';
import BackupIntegrationHealthWidget from './Components/BackupIntegrationHealthWidget';

function SystemHealthDashboard(): ReactElement {
    const [pollingCountFaster, setPollingCountFaster] = useState(0);

    useInterval(() => {
        setPollingCountFaster(pollingCountFaster + 1);
    }, 30000); // 30 seconds is same as for Cluster Status Problems in top navigation

    return (
        <>
            <PageSection variant={PageSectionVariants.light}>
                <Flex>
                    <FlexItem>
                        <Title headingLevel="h1">System Health</Title>
                    </FlexItem>
                    <FlexItem align={{ default: 'alignRight' }}>
                        <GenerateDiagnosticBundle />
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection>
                <Grid hasGutter className="pf-u-mb-sm">
                    {/* TODO: this section migrated as part of
                    https://stack-rox.atlassian.net/browse/ROX-8315 and
                    https://stack-rox.atlassian.net/browse/ROX-8315 */}
                    <GridItem span={12} md={6} lg={8} rowSpan={2}>
                        <Card isCompact>
                            <CardHeader>
                                <CardHeaderMain>
                                    <CardTitle component="h2">Cluster Health</CardTitle>
                                </CardHeaderMain>
                                <CardActions hasNoOffset>
                                    <ViewAllButton url={clustersBasePath} />
                                </CardActions>
                            </CardHeader>
                            <CardBody>grid of smaller cards goes here</CardBody>
                        </Card>
                    </GridItem>
                    <GridItem span={12} md={6} lg={4} rowSpan={1}>
                        {/* TODO: this section migrated as part of https://stack-rox.atlassian.net/browse/ROX-8313 */}
                        <VulnerabilityDefinitionsWidget />
                    </GridItem>
                </Grid>
                <Grid hasGutter>
                    {/* TODO: these section migrated as part of https://stack-rox.atlassian.net/browse/ROX-8314 */}
                    <GridItem span={12} md={4}>
                        <ImageIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                    <GridItem span={12} md={4}>
                        <NotifierIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                    <GridItem span={12} md={4}>
                        <BackupIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
}

export default SystemHealthDashboard;
