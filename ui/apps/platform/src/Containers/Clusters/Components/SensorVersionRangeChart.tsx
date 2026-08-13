import { Flex, FlexItem, Tooltip } from '@patternfly/react-core';

import type { SensorVersionCompatibility } from 'types/cluster.proto';
import { getVersionMajorMinor } from 'utils/versioning';
import { getSensorCompatibilityInfo } from '../cluster.helpers';

export type CompatibilityZone = {
    width: number;
    color: string;
    tooltip: string;
};

type CompatibilityDirection = 'behind' | 'matched' | 'ahead';

export type VersionRangeData = {
    zones: CompatibilityZone[];
    behindVersions: string[];
    matchedVersion: string;
    aheadVersions: string[];
    markerPercent: number | null;
};

const compatibilityRangeColor = {
    incompatible: 'var(--pf-t--global--color--status--danger--default)',
    compatible: 'var(--pf-t--global--color--status--info--default)',
    matched: 'var(--pf-t--global--color--status--success--default)',
    marker: 'var(--pf-t--global--text--color--regular)',
};

function getExpectedDirection(
    compatibility: SensorVersionCompatibility | undefined
): CompatibilityDirection | null {
    switch (compatibility) {
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND':
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND':
            return 'behind';
        case 'SENSOR_VERSION_COMPATIBILITY_MATCHED':
            return 'matched';
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD':
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD':
            return 'ahead';
        default:
            return null;
    }
}

function getActualDirection(sensorIndex: number, centralIndex: number): CompatibilityDirection {
    if (sensorIndex < centralIndex) {
        return 'behind';
    }
    if (sensorIndex > centralIndex) {
        return 'ahead';
    }
    return 'matched';
}

function computeMarkerPercent(
    compatibleVersions: string[],
    sensorVersion: string,
    centralIndex: number,
    compatibility: SensorVersionCompatibility | undefined,
    totalUnits: number
): number | null {
    const toPercent = (position: number) => (position / totalUnits) * 100;

    // Offset: +1 for the left incompatible zone, +0.5 to center within the unit
    const sensorXY = getVersionMajorMinor(sensorVersion);
    if (sensorXY) {
        const sensorIndex = compatibleVersions.indexOf(sensorXY);
        if (sensorIndex !== -1) {
            // The backend compatibility enum is authoritative; discard the
            // parsed position if it falls in a zone that contradicts it.
            const expected = getExpectedDirection(compatibility);
            const actual = getActualDirection(sensorIndex, centralIndex);
            if (!expected || expected === actual) {
                return toPercent(1 + sensorIndex + 0.5);
            }
        }
    }

    const behindCount = centralIndex;
    const aheadCount = compatibleVersions.length - centralIndex - 1;

    // Determines the percentage position from the left edge of the chart to
    // match the appropriate zone.
    switch (compatibility) {
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND':
            return toPercent(0.5);
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND':
            return toPercent(1 + behindCount / 2);
        case 'SENSOR_VERSION_COMPATIBILITY_MATCHED':
            return toPercent(1 + centralIndex + 0.5);
        case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD':
            return toPercent(1 + centralIndex + 1 + aheadCount / 2);
        case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD':
            return toPercent(totalUnits - 0.5);
        default:
            return null;
    }
}

function buildZones(compatibleVersions: string[], centralIndex: number): CompatibilityZone[] {
    const behindVersions = compatibleVersions.slice(0, centralIndex);
    const aheadVersions = compatibleVersions.slice(centralIndex + 1);

    const zones: CompatibilityZone[] = [
        {
            width: 1,
            color: compatibilityRangeColor.incompatible,
            tooltip: `Incompatible sensor versions. < ${compatibleVersions[0]}`,
        },
    ];

    if (behindVersions.length > 0) {
        zones.push({
            width: behindVersions.length,
            color: compatibilityRangeColor.compatible,
            tooltip: `Sensor versions compatible but behind Central. ${behindVersions.join(', ')}`,
        });
    }

    zones.push({
        width: 1,
        color: compatibilityRangeColor.matched,
        tooltip: `Sensor version matched with Central. ${compatibleVersions[centralIndex]}`,
    });

    if (aheadVersions.length > 0) {
        zones.push({
            width: aheadVersions.length,
            color: compatibilityRangeColor.compatible,
            tooltip: `Sensor versions compatible but ahead of Central. ${aheadVersions.join(', ')}`,
        });
    }

    zones.push({
        width: 1,
        color: compatibilityRangeColor.incompatible,
        tooltip: `Incompatible sensor versions. > ${compatibleVersions[compatibleVersions.length - 1]}`,
    });

    return zones;
}

export function computeVersionRangeData(
    compatibleVersions: string[],
    sensorVersion: string,
    centralVersion: string,
    compatibility: SensorVersionCompatibility | undefined
): VersionRangeData | null {
    if (compatibleVersions.length === 0) {
        return null;
    }

    const centralXY = getVersionMajorMinor(centralVersion);
    if (!centralXY) {
        return null;
    }

    const centralIndex = compatibleVersions.indexOf(centralXY);
    if (centralIndex === -1) {
        return null;
    }

    const totalUnits = compatibleVersions.length + 2;

    return {
        zones: buildZones(compatibleVersions, centralIndex),
        behindVersions: compatibleVersions.slice(0, centralIndex),
        matchedVersion: centralXY,
        aheadVersions: compatibleVersions.slice(centralIndex + 1),
        markerPercent: computeMarkerPercent(
            compatibleVersions,
            sensorVersion,
            centralIndex,
            compatibility,
            totalUnits
        ),
    };
}

function buildAriaLabel(
    compatibility: SensorVersionCompatibility | undefined,
    sensorVersion: string,
    centralVersion: string,
    compatibleVersions: string[]
): string {
    const { displayValue } = getSensorCompatibilityInfo(compatibility);
    const range =
        compatibleVersions.length > 0
            ? `${compatibleVersions[0]} to ${compatibleVersions[compatibleVersions.length - 1]}`
            : 'unknown';

    return `Sensor version compatibility chart. Sensor ${sensorVersion || 'unknown'}, Central ${centralVersion}. Status: ${displayValue}. Compatible range: ${range}.`;
}

type SensorVersionRangeChartProps = {
    compatibleVersions: string[];
    sensorVersion: string;
    centralVersion: string;
    compatibility?: SensorVersionCompatibility;
};

function SensorVersionRangeChart({
    compatibleVersions,
    sensorVersion,
    centralVersion,
    compatibility,
}: SensorVersionRangeChartProps) {
    const chartData = computeVersionRangeData(
        compatibleVersions,
        sensorVersion,
        centralVersion,
        compatibility
    );

    if (!chartData) {
        return null;
    }

    const { zones, behindVersions, matchedVersion, aheadVersions, markerPercent } = chartData;
    const { zoneLabel } = getSensorCompatibilityInfo(compatibility);
    const versionLabels = ['', ...behindVersions, matchedVersion, ...aheadVersions, ''];

    return (
        <Flex
            role="img"
            aria-label={buildAriaLabel(
                compatibility,
                sensorVersion,
                centralVersion,
                compatibleVersions
            )}
            direction={{ default: 'column' }}
            gap={{ default: 'gapSm' }}
            className="pf-v6-u-font-size-xs pf-v6-u-text-align-center pf-v6-u-pb-lg"
        >
            <Flex gap={{ default: 'gapNone' }}>
                {versionLabels.map((v, i) => (
                    // eslint-disable-next-line react/no-array-index-key
                    <FlexItem key={i} flex={{ default: 'flex_1' }}>
                        {v}
                    </FlexItem>
                ))}
            </Flex>

            <FlexItem style={{ position: 'relative' }}>
                <Flex gap={{ default: 'gapXs' }}>
                    {zones.map((zone, i) => (
                        <Tooltip
                            // eslint-disable-next-line react/no-array-index-key
                            key={i}
                            content={zone.tooltip}
                        >
                            <FlexItem
                                style={{
                                    flex: zone.width,
                                    height: 16,
                                    backgroundColor: zone.color,
                                }}
                            />
                        </Tooltip>
                    ))}
                </Flex>
                {markerPercent !== null && (
                    <>
                        <div
                            className="pf-v6-u-box-shadow-sm"
                            style={{
                                position: 'absolute',
                                left: `${markerPercent}%`,
                                top: '50%',
                                transform: 'translate(-50%, -50%)',
                                width: 6,
                                height: 30,
                                backgroundColor: compatibilityRangeColor.marker,
                            }}
                        />
                        {zoneLabel && (
                            <div
                                className="pf-v6-u-text-nowrap pf-v6-u-mt-sm"
                                style={{
                                    position: 'absolute',
                                    left: `${markerPercent}%`,
                                    top: '100%',
                                    transform: 'translateX(-50%)',
                                }}
                            >
                                {zoneLabel}
                            </div>
                        )}
                    </>
                )}
            </FlexItem>
        </Flex>
    );
}

export default SensorVersionRangeChart;
