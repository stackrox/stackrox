import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import { SystemConfig } from 'types/config.proto';
import { selectors } from 'reducers';

import PrivateConfigDataRetentionDetails from './PrivateConfigDataRetentionDetails';
import PublicConfigBannerDetails from './PublicConfigBannerDetails';
import PublicConfigLoginDetails from './PublicConfigLoginDetails';
import PublicConfigTelemetryDetails from './PublicConfigTelemetryDetails';

export type SystemConfigDetailsProps = {
    isClustersRoutePathRendered: boolean;
    systemConfig: SystemConfig;
};

function SystemConfigDetails({
    isClustersRoutePathRendered,
    systemConfig,
}: SystemConfigDetailsProps): ReactElement {
    const isTelemetryConfigured = useSelector(selectors.getIsTelemetryConfigured);
    return (
        <>
            <PageSection data-testid="data-retention-config">
                <Title headingLevel="h2" className="pf-u-mb-md">
                    Private data retention configuration
                </Title>
                <PrivateConfigDataRetentionDetails
                    isClustersRoutePathRendered={isClustersRoutePathRendered}
                    privateConfig={systemConfig?.privateConfig}
                />
            </PageSection>
            <PageSection>
                <Title headingLevel="h2" className="pf-u-mb-md">
                    Public configuration
                </Title>
                <Grid hasGutter>
                    <GridItem sm={12} md={6}>
                        <PublicConfigBannerDetails
                            type="header"
                            publicConfig={systemConfig?.publicConfig}
                        />
                    </GridItem>
                    <GridItem sm={12} md={6}>
                        <PublicConfigBannerDetails
                            type="footer"
                            publicConfig={systemConfig?.publicConfig}
                        />
                    </GridItem>
                    <GridItem sm={12} md={6}>
                        <PublicConfigLoginDetails publicConfig={systemConfig?.publicConfig} />
                    </GridItem>
                    {isTelemetryConfigured && (
                        <GridItem sm={12} md={6}>
                            <PublicConfigTelemetryDetails
                                publicConfig={systemConfig?.publicConfig}
                            />
                        </GridItem>
                    )}
                </Grid>
            </PageSection>
        </>
    );
}

export default SystemConfigDetails;
