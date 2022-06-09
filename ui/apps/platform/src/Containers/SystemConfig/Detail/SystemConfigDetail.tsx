import React, { ReactElement } from 'react';
import { Grid, GridItem } from '@patternfly/react-core';

import { SystemConfig } from 'types/config.proto';
import SystemConfigBannerDetail from './SystemConfigBannerDetail';
import SystemConfigLoginDetail from './SystemConfigLoginDetail';
import SystemConfigDataRetentionDetail from './SystemConfigDataRetentionDetail';

export type SystemConfigDetailProps = {
    systemConfig: SystemConfig;
};

function SystemConfigDetail({ systemConfig }: SystemConfigDetailProps): ReactElement {
    return (
        <Grid hasGutter>
            <GridItem span={12}>
                <SystemConfigDataRetentionDetail privateConfig={systemConfig?.privateConfig} />
            </GridItem>
            <GridItem sm={12} md={6}>
                <SystemConfigBannerDetail type="header" publicConfig={systemConfig?.publicConfig} />
            </GridItem>
            <GridItem sm={12} md={6}>
                <SystemConfigBannerDetail type="footer" publicConfig={systemConfig?.publicConfig} />
            </GridItem>
            <GridItem sm={12} md={6}>
                <SystemConfigLoginDetail publicConfig={systemConfig?.publicConfig} />
            </GridItem>
        </Grid>
    );
}

export default SystemConfigDetail;
