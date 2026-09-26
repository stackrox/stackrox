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

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import type { SensorVersionCompatibility } from 'types/cluster.proto';
import { getSensorCompatibilityInfo, shouldShowSensorVersionRangeChart } from '../cluster.helpers';
import SensorVersionRangeChart from './SensorVersionRangeChart';

// TODO: replace with actual documentation URL
const COMPATIBILITY_DOC_LINK = '';

const documentationLink = (
    <ExternalLink>
        <a href={COMPATIBILITY_DOC_LINK} target="_blank" rel="noopener noreferrer">
            See documentation
        </a>
    </ExternalLink>
);

function getGuidanceText(compatibility: SensorVersionCompatibility | undefined): ReactNode {
    switch (compatibility) {
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND':
            return (
                <>
                    Match Sensor to Central version, or at minimum to be within the compatible
                    version range. {documentationLink}.
                </>
            );
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND':
            return (
                <>
                    No immediate action is required. It is recommended to match Sensor and Central
                    versions for optimal functionality. {documentationLink}.
                </>
            );
        case 'SENSOR_VERSION_COMPATIBILITY_MATCHED':
            return '';
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD':
            return (
                <>
                    No immediate action is required. It is recommended to match Sensor and Central
                    versions for optimal functionality. {documentationLink}.
                </>
            );
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD':
            return (
                <>
                    Match Sensor to Central version, or at minimum to be within the compatible
                    version range. {documentationLink}.
                </>
            );
        case 'SENSOR_VERSION_COMPATIBILITY_UNKNOWN':
            return '-';
        default:
            return '-';
    }
}

function getSensorInformationText(compatibility: SensorVersionCompatibility | undefined): string {
    switch (compatibility) {
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND':
            return 'Sensor version is outside the compatible version range and is behind Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND':
            return 'Sensor version is compatible with Central but is behind Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_MATCHED':
            return 'Sensor version matched Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD':
            return 'Sensor version is compatible with Central but is ahead of Central.';
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD':
            return 'Sensor version is outside the compatible version range and is ahead of Central.';
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
                            {Object.entries(bodyItems).map(
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

export type SensorCompatibilityPanelProps = {
    compatibility?: SensorVersionCompatibility;
    compatibleVersions: string[];
    sensorVersion?: string;
    centralVersion: string;
};

function SensorCompatibilityPanel({
    compatibility,
    compatibleVersions,
    sensorVersion,
    centralVersion,
}: SensorCompatibilityPanelProps) {
    const { displayValue, Icon, fgColor } = getSensorCompatibilityInfo(compatibility);

    const showGuidance =
        typeof compatibility === 'string' &&
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
                    status: ['Status', displayValue],
                    info: ['Status information', getSensorInformationText(compatibility)],
                    guidance: ['Guidance', showGuidance && getGuidanceText(compatibility)],
                }}
            />
            <SensorCompatibilitySubPanel
                header="Sensor version"
                bodyItems={{
                    version: ['Version', sensorVersion ?? 'Unknown'],
                    range: [
                        'Version range',
                        shouldShowSensorVersionRangeChart(compatibility, compatibleVersions) ? (
                            <SensorVersionRangeChart
                                compatibleVersions={compatibleVersions}
                                sensorVersion={sensorVersion ?? ''}
                                centralVersion={centralVersion}
                                compatibility={compatibility}
                            />
                        ) : null,
                    ],
                }}
            />
            <SensorCompatibilitySubPanel
                header="Central version"
                bodyItems={{
                    version: ['Version', centralVersion],
                    compatibleVersions: [
                        <>
                            Compatible sensor versions{' '}
                            <Tooltip content="Future Sensor versions are subject to change based on Red Hat's release planning.">
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
