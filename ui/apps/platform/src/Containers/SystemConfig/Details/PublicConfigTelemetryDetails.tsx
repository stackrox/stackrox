import React, { ReactElement } from 'react';

import { PublicConfig } from 'types/config.proto';
import {
    Card,
    CardActions,
    CardBody,
    CardHeader,
    CardHeaderMain,
    CardTitle,
    Divider,
    Label,
} from '@patternfly/react-core';

export type PublicConfigTelemetryDetailsProps = {
    publicConfig: PublicConfig | null;
};

const PublicConfigTelemetryDetails = ({
    publicConfig,
}: PublicConfigTelemetryDetailsProps): ReactElement => {
    const isEnabled = publicConfig?.telemetry?.enabled || false;

    return (
        <Card isFlat data-testid="telemetry-config">
            <CardHeader>
                <CardHeaderMain>
                    <CardTitle component="h3">Telemetry configuration</CardTitle>
                </CardHeaderMain>
                <CardActions data-testid="telemetry-state">
                    {isEnabled ? <Label color="green">Enabled</Label> : <Label>Disabled</Label>}
                </CardActions>
            </CardHeader>
            <Divider component="div" />
            <CardBody>
                <p className="pf-u-mb-sm">
                    Online telemetry data collection allows Red Hat to use anonymized information to
                    enhance your user experience.
                </p>
            </CardBody>
        </Card>
    );
};

export default PublicConfigTelemetryDetails;
