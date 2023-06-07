import React, { useState } from 'react';
import { Flex, FlexItem, Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import useInterval from 'hooks/useInterval';
import useCentralCapabilities from 'hooks/useCentralCapabilities';

import CertificateCard from './CertificateHealth/CertificateCard';
import ClustersHealthCards from './ClustersHealth/ClustersHealthCards';
import DeclarativeConfigurationHealthCard from './DeclarativeConfigurationHealth/DeclarativeConfigurationHealthCard';
import GenerateDiagnosticBundle from './DiagnosticBundle/GenerateDiagnosticBundle';
import VulnerabilityDefinitionsHealthCard from './VulnerabilityDefinitionsHealth/VulnerabilityDefinitionsHealthCard';

import ImageIntegrationHealthWidget from './Components/ImageIntegrationHealthWidget';
import NotifierIntegrationHealthWidget from './Components/NotifierIntegrationHealthWidget';
import BackupIntegrationHealthWidget from './Components/BackupIntegrationHealthWidget';

const SystemHealthDashboardPage = () => {
    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const isDeclarativeConfigHealthAvailable = isCentralCapabilityAvailable(
        'centralCanDisplayDeclarativeConfigHealth'
    );

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
                    <ClustersHealthCards pollingCount={pollingCountFaster} />
                    <GridItem span={12}>
                        <VulnerabilityDefinitionsHealthCard pollingCount={pollingCountSlower} />
                    </GridItem>
                    <GridItem span={12}>
                        <ImageIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                    <GridItem span={12}>
                        <NotifierIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                    <GridItem span={12}>
                        <BackupIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    </GridItem>
                    {isDeclarativeConfigHealthAvailable && (
                        <GridItem span={12}>
                            <DeclarativeConfigurationHealthCard pollingCount={pollingCountFaster} />
                        </GridItem>
                    )}
                    <GridItem span={12}>
                        <CertificateCard component="CENTRAL" pollingCount={pollingCountSlower} />
                    </GridItem>
                    <GridItem span={12}>
                        <CertificateCard component="SCANNER" pollingCount={pollingCountSlower} />
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
};

export default SystemHealthDashboardPage;
