import React, { useEffect, useState } from 'react';
import { Alert, Flex, FlexItem, Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import Widget from 'Components/Widget';
import ViewAllButton from 'Components/ViewAllButton';
import useInterval from 'hooks/useInterval';
import { clustersBasePath } from 'routePaths';
import { fetchClustersAsArray } from 'services/ClustersService';

import ClusterOverview from './Components/ClusterOverview';
import CollectorStatus from './Components/CollectorStatus';
import AdmissionControlStatus from './Components/AdmissionControlStatus';
import CredentialExpiration from './Components/CredentialExpiration';
import DeclarativeConfigurationHealthCard from './Components/DeclarativeConfigurationHealthCard';
import GenerateDiagnosticBundle from './Components/GenerateDiagnosticBundle';
import SensorStatus from './Components/SensorStatus';
import SensorUpgrade from './Components/SensorUpgrade';
import VulnerabilityDefinitionsHealthCard from './Components/VulnerabilityDefinitionsHealthCard';
import ImageIntegrationHealthWidget from './Components/ImageIntegrationHealthWidget';
import NotifierIntegrationHealthWidget from './Components/NotifierIntegrationHealthWidget';
import BackupIntegrationHealthWidget from './Components/BackupIntegrationHealthWidget';

const SystemHealthDashboardPage = () => {
    const [pollingCountFaster, setPollingCountFaster] = useState(0);
    const [pollingCountSlower, setPollingCountSlower] = useState(0);
    const [currentDatetime, setCurrentDatetime] = useState(null);

    const [clusters, setClusters] = useState([]);

    const [clustersRequestHasError, setClustersRequestHasError] = useState(false);

    useEffect(() => {
        setCurrentDatetime(new Date());
        fetchClustersAsArray()
            .then((array) => {
                setClusters(array);
                setClustersRequestHasError(false);
            })
            .catch(() => {
                setClusters([]);
                setClustersRequestHasError(true);
            });
    }, [pollingCountFaster]);

    useInterval(() => {
        setPollingCountFaster(pollingCountFaster + 1);
    }, 30000); // 30 seconds is same as for Cluster Status Problems in top navigation

    useInterval(() => {
        setPollingCountSlower(pollingCountSlower + 1);
    }, 300000); // 5 minutes is enough for Vulnerability Definitions

    return (
        <>
            <PageSection variant="light">
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
                <Grid hasGutter>
                    <GridItem span={12} rowSpan={2}>
                        <Widget
                            className="sx-2 theme-light"
                            header="Cluster Health"
                            headerComponents={<ViewAllButton url={clustersBasePath} />}
                            id="cluster-health"
                        >
                            {clustersRequestHasError ? (
                                <Alert
                                    variant="warning"
                                    isInline
                                    className="pf-u-w-100"
                                    title="Request failed for Clusters"
                                />
                            ) : (
                                <div className="flex flex-wrap">
                                    <Widget
                                        className="h-48 m-2 w-48 text-lg"
                                        header="Cluster Overview"
                                        id="cluster-overview"
                                    >
                                        <ClusterOverview clusters={clusters} />
                                    </Widget>
                                    <Widget
                                        className="h-48 m-2 text-center w-48 text-lg"
                                        header="Collector Status"
                                        id="collector-status"
                                    >
                                        <CollectorStatus clusters={clusters} />
                                    </Widget>
                                    <Widget
                                        className="h-48 m-2 text-center w-48 text-lg"
                                        header="Admission Control Status"
                                        id="admissionControl-status"
                                    >
                                        <AdmissionControlStatus clusters={clusters} />
                                    </Widget>
                                    <Widget
                                        className="h-48 m-2 text-center w-48 text-lg"
                                        header="Sensor Status"
                                        id="sensor-status"
                                    >
                                        <SensorStatus clusters={clusters} />
                                    </Widget>
                                    <Widget
                                        className="h-48 m-2 text-center w-48 text-lg"
                                        header="Sensor Upgrade"
                                        id="sensor-upgrade"
                                    >
                                        <SensorUpgrade clusters={clusters} />
                                    </Widget>
                                    <Widget
                                        className="h-48 m-2 text-center w-48 text-lg"
                                        header="Credential Expiration"
                                        id="credential-expiration"
                                    >
                                        <CredentialExpiration
                                            clusters={clusters}
                                            currentDatetime={currentDatetime}
                                        />
                                    </Widget>
                                </div>
                            )}
                        </Widget>
                    </GridItem>
                    <GridItem span={12}>
                        <VulnerabilityDefinitionsHealthCard pollingCount={pollingCountSlower} />
                    </GridItem>
                    <GridItem span={4}>
                        <ImageIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                    <GridItem span={4}>
                        <NotifierIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                    <GridItem span={4}>
                        <BackupIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                    <GridItem span={12}>
                        <DeclarativeConfigurationHealthCard pollingCount={pollingCountFaster} />
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
};

export default SystemHealthDashboardPage;
