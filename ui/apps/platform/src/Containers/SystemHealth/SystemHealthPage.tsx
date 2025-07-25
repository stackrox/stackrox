import React, { useState } from 'react';
import { Flex, FlexItem, Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import useCentralCapabilities from 'hooks/useCentralCapabilities';
import useInterval from 'hooks/useInterval';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';
import usePermissions from 'hooks/usePermissions';

import CertificateCard from './CertificateHealth/CertificateCard';
import ClustersHealthCards from './ClustersHealth/ClustersHealthCards';
import DeclarativeConfigurationHealthCard from './DeclarativeConfigurationHealth/DeclarativeConfigurationHealthCard';
import GenerateDiagnosticBundle from './DiagnosticBundle/GenerateDiagnosticBundle';
import VulnerabilityDefinitionsHealthCard from './VulnerabilityDefinitionsHealth/VulnerabilityDefinitionsHealthCard';

import ImageIntegrationHealthWidget from './Components/ImageIntegrationHealthWidget';
import NotifierIntegrationHealthWidget from './Components/NotifierIntegrationHealthWidget';
import BackupIntegrationHealthWidget from './Components/BackupIntegrationHealthWidget';
import ShowAdministrationUsage from './UsageStatistics/ShowAdministrationUsage';
import CentralDatabaseHealthCard from './CentralDatabaseHealth/CentralDatabaseHealthCard';

function SystemHealthPage() {
    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const isDeclarativeConfigHealthAvailable = isCentralCapabilityAvailable(
        'centralCanDisplayDeclarativeConfigHealth'
    );

    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAdministration = hasReadAccess('Administration');
    const hasReadAccessForCluster = hasReadAccess('Cluster');
    const hasReadAccessForIntegration = hasReadAccess('Integration');

    const isScannerV4Enabled = useIsScannerV4Enabled();

    const [pollingCountFaster, setPollingCountFaster] = useState(0);
    const [pollingCountSlower, setPollingCountSlower] = useState(0);

    useInterval(() => {
        setPollingCountFaster(pollingCountFaster + 1);
    }, 30000); // 30 seconds is same as for Cluster Status Problems in top navigation

    useInterval(() => {
        setPollingCountSlower(pollingCountSlower + 1);
    }, 300000); // 5 minutes is enough for Vulnerability Definitions

    return (
        <>
            <PageSection variant="light" component="div">
                <Flex>
                    <FlexItem>
                        <Title headingLevel="h1">System Health</Title>
                    </FlexItem>
                    {hasReadAccessForAdministration && (
                        <FlexItem align={{ default: 'alignRight' }}>
                            <Flex>
                                <FlexItem>
                                    <ShowAdministrationUsage />
                                </FlexItem>
                                <FlexItem>
                                    <GenerateDiagnosticBundle />
                                </FlexItem>
                            </Flex>
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <PageSection>
                <Grid hasGutter>
                    {hasReadAccessForCluster && (
                        <ClustersHealthCards pollingCount={pollingCountFaster} />
                    )}
                    <GridItem span={12}>
                        <CentralDatabaseHealthCard />
                    </GridItem>
                    {hasReadAccessForAdministration && (
                        <GridItem span={12}>
                            <VulnerabilityDefinitionsHealthCard
                                component="SCANNER"
                                pollingCount={pollingCountSlower}
                            />
                        </GridItem>
                    )}
                    {isScannerV4Enabled && hasReadAccessForAdministration && (
                        <GridItem span={12}>
                            <VulnerabilityDefinitionsHealthCard
                                component="SCANNER_V4"
                                pollingCount={pollingCountSlower}
                            />
                        </GridItem>
                    )}
                    {hasReadAccessForIntegration && (
                        <>
                            <GridItem span={12}>
                                <ImageIntegrationHealthWidget pollingCount={pollingCountFaster} />
                            </GridItem>
                            <GridItem span={12}>
                                <NotifierIntegrationHealthWidget
                                    pollingCount={pollingCountFaster}
                                />
                            </GridItem>
                            <GridItem span={12}>
                                <BackupIntegrationHealthWidget pollingCount={pollingCountFaster} />
                            </GridItem>
                        </>
                    )}
                    {hasReadAccessForIntegration && isDeclarativeConfigHealthAvailable && (
                        <GridItem span={12}>
                            <DeclarativeConfigurationHealthCard pollingCount={pollingCountFaster} />
                        </GridItem>
                    )}
                    <GridItem span={12}>
                        <CertificateCard component="CENTRAL" pollingCount={pollingCountSlower} />
                    </GridItem>
                    <GridItem span={12}>
                        <CertificateCard component="CENTRAL_DB" pollingCount={pollingCountSlower} />
                    </GridItem>
                    <GridItem span={12}>
                        <CertificateCard component="SCANNER" pollingCount={pollingCountSlower} />
                    </GridItem>
                    {isScannerV4Enabled && (
                        <GridItem span={12}>
                            <CertificateCard
                                component="SCANNER_V4"
                                pollingCount={pollingCountSlower}
                            />
                        </GridItem>
                    )}
                </Grid>
            </PageSection>
        </>
    );
}

export default SystemHealthPage;
