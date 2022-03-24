import React, { ReactElement } from 'react';
import { Grid, GridItem } from '@patternfly/react-core';

import { getProductBranding } from 'constants/productBranding';
import { SystemConfig, TelemetryConfig } from 'Containers/SystemConfig/SystemConfigTypes';
import ConfigBannerDetailWidget from './ConfigBannerDetailWidget';
import ConfigLoginDetailWidget from './ConfigLoginDetailWidget';
import ConfigDataRetentionDetailWidget from './ConfigDataRetentionDetailWidget';
import ConfigTelemetryDetailWidget from './ConfigTelemetryDetailWidget';

export type DetailsProps = {
    systemConfig: SystemConfig;
    telemetryConfig: TelemetryConfig;
};

function Details({ systemConfig, telemetryConfig }: DetailsProps): ReactElement {
    const { type } = getProductBranding();
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
            {type === 'RHACS_BRANDING' && (
                <GridItem sm={12} md={6}>
                    <ConfigTelemetryDetailWidget
                        telemetryConfig={telemetryConfig}
                        editable={false}
                    />
                </GridItem>
            )}
        </Grid>
    );
}

export default Details;
