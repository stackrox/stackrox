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

import ViewAllButton from 'Components/ViewAllButton';
import { clustersBasePath } from 'routePaths';
import GenerateDiagnosticBundleButton from '../Components/GenerateDiagnosticBundleButton';

function SystemHealthDashboard(): ReactElement {
    return (
        <>
            <PageSection variant={PageSectionVariants.light}>
                <Flex>
                    <FlexItem>
                        <Title headingLevel="h1">System Health</Title>
                    </FlexItem>
                    <FlexItem align={{ default: 'alignRight' }}>
                        <GenerateDiagnosticBundleButton />
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection>
                <Grid hasGutter className="pf-u-mb-sm">
                    <GridItem span={12} md={6} lg={8} rowSpan={2}>
                        <Card isCompact>
                            <CardHeader>
                                <CardHeaderMain>
                                    <CardTitle component="h2">Cluster Health</CardTitle>
                                </CardHeaderMain>
                                <CardActions>
                                    <ViewAllButton url={clustersBasePath} />
                                </CardActions>
                            </CardHeader>
                            <CardBody>grid of smaller cards goes here</CardBody>
                        </Card>
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
}

export default SystemHealthDashboard;
