import React, { ReactElement } from 'react';
import { Grid, GridItem } from '@patternfly/react-core';

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
    return (
        <Grid hasGutter>
            <GridItem span={12}>
                <ConfigDataRetentionDetailWidget config={systemConfig} />
            </GridItem>
            <GridItem span={6}>
                <ConfigBannerDetailWidget type="header" config={systemConfig} />
            </GridItem>
            <GridItem span={6}>
                <ConfigBannerDetailWidget type="footer" config={systemConfig} />
            </GridItem>
            <GridItem span={6}>
                <ConfigLoginDetailWidget config={systemConfig} />
            </GridItem>
            <GridItem span={6}>
                <ConfigTelemetryDetailWidget config={telemetryConfig} editable={false} />
            </GridItem>
        </Grid>
    );
}

export default Details;
