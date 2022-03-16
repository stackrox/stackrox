import React, { ReactElement, useState } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardHeaderMain,
    CardTitle,
    CardActions,
    ExpandableSection,
    Label,
    Divider,
} from '@patternfly/react-core';

import ReduxToggleField from 'Components/forms/ReduxToggleField';
import { TelemetryConfig } from 'Containers/SystemConfig/SystemConfigTypes';

export type ConfigTelemetryDetailWidgetProps = {
    telemetryConfig: TelemetryConfig;
    editable: boolean;
};

function getTextOrToggle(telemetryConfig, editable) {
    const isEnabled = telemetryConfig?.enabled || false;
    if (editable) {
        return <ReduxToggleField name="telemetryConfig.enabled" />;
    }
    return isEnabled ? (
        <Label color="green" data-testid="telemetry-state">
            Enabled
        </Label>
    ) : (
        <Label data-testid="telemetry-state">Disabled</Label>
    );
}

export const ConfigTelemetryDetailContent = (): ReactElement => {
    const [isExpanded, setIsExpanded] = useState(false);
    function onToggle() {
        setIsExpanded(!isExpanded);
    }
    return (
        <>
            <p className="pf-u-mb-sm">
                Online telemetry data collection allows Red Hat to use anonymized information to
                enhance your user experience.
            </p>
            <ExpandableSection
                toggleText={isExpanded ? 'Show Less' : 'Show More'}
                onToggle={onToggle}
                isExpanded={isExpanded}
            >
                <p>
                    By consenting to online telemetry data collection, you allow Red Hat to store
                    and perform analytics on data that arises from the usage and operation of Red
                    Hat Advanced Cluster Security. This data may contain both operational metrics of
                    the platform itself, as well as information about the environments in which it
                    is being used. While the data is associated with your account, it does not
                    include any information about the purpose of these environments; in particular,
                    no identifying information like names of nodes, workloads or non-default
                    namespaces.
                </p>
                <p className="pf-u-mt-md">
                    You can revoke your consent to online telemetry data collection at any time. If
                    you wish to request the deletion of already collected data, please contact our
                    Customer Success team.
                </p>
            </ExpandableSection>
        </>
    );
};

const ConfigTelemetryDetailWidget = ({
    telemetryConfig,
    editable,
}: ConfigTelemetryDetailWidgetProps): ReactElement => {
    return (
        <Card>
            <CardHeader>
                <CardHeaderMain>
                    <CardTitle>Online Telemetry Data Collection</CardTitle>
                </CardHeaderMain>
                <CardActions>{getTextOrToggle(telemetryConfig, editable)}</CardActions>
            </CardHeader>
            <Divider component="div" />
            <CardBody>
                <ConfigTelemetryDetailContent />
            </CardBody>
        </Card>
    );
};

export default ConfigTelemetryDetailWidget;
