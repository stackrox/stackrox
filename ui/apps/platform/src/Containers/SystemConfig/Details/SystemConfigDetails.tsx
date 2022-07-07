import React, { ReactElement } from 'react';
import { Grid, GridItem, PageSection, Title } from '@patternfly/react-core';

import { SystemConfig } from 'types/config.proto';

import PrivateConfigDataRetentionDetails from './PrivateConfigDataRetentionDetails';
import PublicConfigBannerDetails from './PublicConfigBannerDetails';
import PublicConfigLoginDetails from './PublicConfigLoginDetails';

export type SystemConfigDetailsProps = {
    isClustersRoutePathRendered: boolean;
    isDecommissionedClusterRetentionEnabled: boolean;
    systemConfig: SystemConfig;
};

function SystemConfigDetails({
    isClustersRoutePathRendered,
    isDecommissionedClusterRetentionEnabled,
    systemConfig,
}: SystemConfigDetailsProps): ReactElement {
    return (
        <>
            <PageSection data-testid="data-retention-config">
                <Title headingLevel="h2" className="pf-u-mb-md">
                    Private data retention configuration
                </Title>
                <PrivateConfigDataRetentionDetails
                    isClustersRoutePathRendered={isClustersRoutePathRendered}
                    isDecommissionedClusterRetentionEnabled={
                        isDecommissionedClusterRetentionEnabled
                    }
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
                </Grid>
            </PageSection>
        </>
    );
}

export default SystemConfigDetails;
