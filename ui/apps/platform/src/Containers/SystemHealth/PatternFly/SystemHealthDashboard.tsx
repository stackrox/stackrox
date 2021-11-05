import React, { ReactElement } from 'react';
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
import { clustersBasePath } from 'routePaths';
import VulnerabilityDefinitionsWidget from './Components/VulnerabilityDefinitionsWidget';
import GenerateDiagnosticBundle from './Components/GenerateDiagnosticBundle';

function SystemHealthDashboard(): ReactElement {
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
                        <Card isCompact>
                            <CardHeader>
                                <CardHeaderMain>
                                    <CardTitle component="h2">Image Integrations</CardTitle>
                                </CardHeaderMain>
                                <CardActions hasNoOffset>
                                    <ViewAllButton url={clustersBasePath} />
                                </CardActions>
                            </CardHeader>
                            <CardBody>health details go here</CardBody>
                        </Card>
                    </GridItem>
                    <GridItem span={12} md={4}>
                        <Card isCompact>
                            <CardHeader>
                                <CardHeaderMain>
                                    <CardTitle component="h2">Notifier Integrations</CardTitle>
                                </CardHeaderMain>
                                <CardActions hasNoOffset>
                                    <ViewAllButton url={clustersBasePath} />
                                </CardActions>
                            </CardHeader>
                            <CardBody>health details go here</CardBody>
                        </Card>
                    </GridItem>
                    <GridItem span={12} md={4}>
                        <Card isCompact>
                            <CardHeader>
                                <CardHeaderMain>
                                    <CardTitle component="h2">Backup Integrations</CardTitle>
                                </CardHeaderMain>
                                <CardActions hasNoOffset>
                                    <ViewAllButton url={clustersBasePath} />
                                </CardActions>
                            </CardHeader>
                            <CardBody>health details go here</CardBody>
                        </Card>
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
}

export default SystemHealthDashboard;
