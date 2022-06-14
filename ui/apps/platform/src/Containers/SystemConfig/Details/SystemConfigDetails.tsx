import React, { ReactElement } from 'react';
import { Grid, GridItem } from '@patternfly/react-core';

import { SystemConfig } from 'types/config.proto';

import PrivateConfigDataRetentionDetails from './PrivateConfigDataRetentionDetails';
import PublicConfigBannerDetails from './PublicConfigBannerDetails';
import PublicConfigLoginDetails from './PublicConfigLoginDetails';

export type SystemConfigDetailProps = {
    systemConfig: SystemConfig;
};

function SystemConfigDetail({ systemConfig }: SystemConfigDetailProps): ReactElement {
    return (
        <Grid hasGutter>
            <GridItem span={12}>
                <PrivateConfigDataRetentionDetails privateConfig={systemConfig?.privateConfig} />
            </GridItem>
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
    );
}

export default SystemConfigDetail;
