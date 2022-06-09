import React, { ReactElement } from 'react';
import { Grid, GridItem } from '@patternfly/react-core';

import { SystemConfig } from 'types/config.proto';
import ConfigBannerDetailWidget from './ConfigBannerDetailWidget';
import ConfigLoginDetailWidget from './ConfigLoginDetailWidget';
import ConfigDataRetentionDetailWidget from './ConfigDataRetentionDetailWidget';

export type DetailsProps = {
    systemConfig: SystemConfig;
};

function Details({ systemConfig }: DetailsProps): ReactElement {
    return (
        <Grid hasGutter>
            <GridItem span={12}>
                <ConfigDataRetentionDetailWidget privateConfig={systemConfig?.privateConfig} />
            </GridItem>
            <GridItem sm={12} md={6}>
                <ConfigBannerDetailWidget type="header" publicConfig={systemConfig?.publicConfig} />
            </GridItem>
            <GridItem sm={12} md={6}>
                <ConfigBannerDetailWidget type="footer" publicConfig={systemConfig?.publicConfig} />
            </GridItem>
            <GridItem sm={12} md={6}>
                <ConfigLoginDetailWidget publicConfig={systemConfig?.publicConfig} />
            </GridItem>
        </Grid>
    );
}

export default Details;
