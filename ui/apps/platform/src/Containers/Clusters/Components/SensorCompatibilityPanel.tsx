import type { ReactNode } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    Grid,
    GridItem,
    Panel,
    PanelHeader,
    PanelMain,
    PanelMainBody,
    Tooltip,
} from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

import type { SensorVersionCompatibility } from 'types/cluster.proto';
import {
    getSensorCompatibilityDisplayValue,
    getSensorCompatibilityStyle,
} from '../cluster.helpers';

export type SensorCompatibilityPanelProps = {
    compatibility: SensorVersionCompatibility;
    compatibleVersions: string[];
    sensorVersion: string;
    centralVersion: string;
};

// TODO: add documentation link for guidance text
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const COMPATIBILITY_DOC_LINK = '';

function getGuidanceText(compatibility: SensorVersionCompatibility): string {
    switch (compatibility) {
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND':
            return 'No immediate action is required. Upgrade Sensor to match Central for optimal functionality. See documentation.';
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD':
            return 'No immediate action is required. It is recommended to plan a Central upgrade taking into account versions of all connected Sensors. If you prefer not to upgrade Central, suggest downgrading Sensor to match Central as the alternative option. See documentation.';
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND':
            return 'Upgrade Sensor to match Central, or at minimum to within the compatible version range. See documentation.';
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD':
            return 'Plan a Central upgrade taking into account versions of all connected Sensors or downgrade Sensor to match Central or at least to be within the compatible version range with Central. See documentation.';
        case 'SENSOR_VERSION_COMPATIBILITY_MATCHED':
            return '';
        case 'SENSOR_VERSION_COMPATIBILITY_UNKNOWN':
            return '-';
        default:
            return '-';
    }
}

function getSensorInformationText(compatibility: SensorVersionCompatibility): string {
    switch (compatibility) {
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND':
            return 'Sensor version is compatible with Central but is behind Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD':
            return 'Sensor version is compatible with Central but is ahead of Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND':
            return 'Sensor version is outside the compatible version range and is behind Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD':
            return 'Sensor version is outside the compatible version range and is ahead of Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_MATCHED':
            return 'Sensor version is matched with Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_UNKNOWN':
            return '-';
        default:
            return '-';
    }
}

function SensorCompatibilitySubPanel({
    header,
    bodyItems,
}: {
    header: ReactNode;
    bodyItems: Record<string, [ReactNode, ReactNode | null]>;
}) {
    return (
        <GridItem span={12} xl2={4}>
            {/* TODO PatternFly 6.6.0 has an `isFullHeight` prop we can use instead */}
            <Panel variant="bordered" className="pf-v6-u-h-100">
                <PanelHeader>{header}</PanelHeader>
                <Divider />
                <PanelMain>
                    <PanelMainBody>
                        <DescriptionList>
                            {Object.entries(bodyItems).flatMap(
                                ([key, [term, description]]) =>
                                    description && (
                                        <DescriptionListGroup key={key}>
                                            <DescriptionListTerm>{term}</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {description}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                    )
                            )}
                        </DescriptionList>
                    </PanelMainBody>
                </PanelMain>
            </Panel>
        </GridItem>
    );
}

function SensorCompatibilityPanel({
    compatibility,
    compatibleVersions,
    sensorVersion,
    centralVersion,
}: SensorCompatibilityPanelProps) {
    const { Icon, fgColor } = getSensorCompatibilityStyle(compatibility);

    const showGuidance =
        compatibility !== 'SENSOR_VERSION_COMPATIBILITY_MATCHED' &&
        compatibility !== 'SENSOR_VERSION_COMPATIBILITY_UNKNOWN';

    return (
        <Grid hasGutter>
            <SensorCompatibilitySubPanel
                header={
                    <Flex alignItems={{ default: 'alignItemsCenter' }} gap={{ default: 'gapSm' }}>
                        <Icon className={fgColor} />
                        <span>Sensor status</span>
                    </Flex>
                }
                bodyItems={{
                    status: ['Status', getSensorCompatibilityDisplayValue(compatibility)],
                    info: ['Status information', getSensorInformationText(compatibility)],
                    guidance: ['Guidance', showGuidance && getGuidanceText(compatibility)],
                }}
            />
            <SensorCompatibilitySubPanel
                header="Sensor version"
                bodyItems={{
                    version: ['Version', sensorVersion],
                    range: ['Version range', null],
                }}
            />
            <SensorCompatibilitySubPanel
                header="Central version"
                bodyItems={{
                    version: ['Version', centralVersion],
                    compatibleVersions: [
                        <>
                            Compatible sensor versions{' '}
                            <Tooltip content="Future Sensor version is subject to change based on Red Hat's release planning.">
                                <HelpIcon />
                            </Tooltip>
                        </>,
                        compatibleVersions.join(', '),
                    ],
                }}
            />
        </Grid>
    );
}

export default SensorCompatibilityPanel;
