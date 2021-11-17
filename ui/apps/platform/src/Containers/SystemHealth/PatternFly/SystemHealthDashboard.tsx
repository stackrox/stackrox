import React, { ReactElement, useEffect, useState } from 'react';
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
import { fetchVulnerabilityDefinitionsInfo } from 'services/IntegrationHealthService';
import VulnerabilityDefinitionsWidget from './Components/VulnerabilityDefinitionsWidget';
import GenerateDiagnosticBundle from './Components/GenerateDiagnosticBundle';
import ImageIntegrationHealthWidget from './Components/ImageIntegrationHealthWidget';
import NotifierIntegrationHealthWidget from './Components/NotifierIntegrationHealthWidget';
import BackupIntegrationHealthWidget from './Components/BackupIntegrationHealthWidget';

function SystemHealthDashboard(): ReactElement {
    const [pollingCountFaster, setPollingCountFaster] = useState(0);
    const [pollingCountSlower, setPollingCountSlower] = useState(0);
    const [currentDatetime, setCurrentDatetime] = useState<Date | null>(null);

    const [vulnerabilityDefinitionsInfo, setVulnerabilityDefinitionsInfo] = useState(null);

    const [vulnerabilityDefinitionsRequestHasError, setVulnerabilityDefinitionsRequestHasError] =
        useState(false);

    useEffect(() => {
        setCurrentDatetime(new Date());
    }, [pollingCountFaster]);

    useEffect(() => {
        fetchVulnerabilityDefinitionsInfo()
            .then((info) => {
                setVulnerabilityDefinitionsInfo(info);
                setVulnerabilityDefinitionsRequestHasError(false);
            })
            .catch(() => {
                setVulnerabilityDefinitionsInfo(null);
                setVulnerabilityDefinitionsRequestHasError(true);
            });
    }, [pollingCountSlower]);

    useInterval(() => {
        setPollingCountFaster(pollingCountFaster + 1);
    }, 30000); // 30 seconds is same as for Cluster Status Problems in top navigation

    useInterval(() => {
        setPollingCountSlower(pollingCountSlower + 1);
    }, 300000); // 5 minutes is enough for Vulnerability Definitions

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
                        <VulnerabilityDefinitionsWidget
                            currentDatetime={currentDatetime}
                            vulnerabilityDefinitionsInfo={vulnerabilityDefinitionsInfo}
                            hasError={vulnerabilityDefinitionsRequestHasError}
                        />
                    </GridItem>
                </Grid>
                <Grid hasGutter>
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
