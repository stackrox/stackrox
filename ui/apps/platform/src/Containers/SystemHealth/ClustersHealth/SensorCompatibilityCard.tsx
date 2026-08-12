import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import {
    Alert,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    pluralize,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import IconText from 'Components/PatternFly/IconText/IconText';
import { sensorCompatibilityMap } from 'Containers/Clusters/cluster.helpers';
import type { SensorCompatibilityInfo } from 'Containers/Clusters/cluster.helpers';
import { clustersBasePath } from 'routePaths';
import type { Cluster } from 'types/cluster.proto';

import { ErrorIcon, SpinnerIcon, healthIconMap } from '../CardHeaderIcons';
import type { HealthVariant } from '../CardHeaderIcons';
import { TdTotal } from './ClustersHealthTable';

const dataLabelMatched = 'Matched';
const dataLabelCompatible = 'Compatible';
const dataLabelIncompatible = 'Incompatible';
const dataLabelUnknown = 'Unknown';

const compatibilityStyles = {
    MATCHED: sensorCompatibilityMap.SENSOR_VERSION_COMPATIBILITY_MATCHED,
    COMPATIBLE: sensorCompatibilityMap.SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND,
    INCOMPATIBLE: sensorCompatibilityMap.SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND,
    UNKNOWN: sensorCompatibilityMap.SENSOR_VERSION_COMPATIBILITY_UNKNOWN,
} as const;

export type SensorCompatibilityCounts = Record<
    'MATCHED' | 'COMPATIBLE' | 'INCOMPATIBLE' | 'UNKNOWN',
    number
>;

export function getSensorCompatibilityCounts(clusters: Cluster[]): SensorCompatibilityCounts {
    const counts = {
        MATCHED: 0,
        COMPATIBLE: 0,
        INCOMPATIBLE: 0,
        UNKNOWN: 0,
    };

    clusters.forEach((cluster) => {
        if (!cluster || !cluster.status) {
            counts.UNKNOWN += 1;
            return;
        }
        const { sensorVersionCompatibility } = cluster.status;

        switch (sensorVersionCompatibility) {
            case 'SENSOR_VERSION_COMPATIBILITY_MATCHED':
                counts.MATCHED += 1;
                break;
            case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND':
            case 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD':
                counts.COMPATIBLE += 1;
                break;
            case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND':
            case 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD':
                counts.INCOMPATIBLE += 1;
                break;
            default:
                counts.UNKNOWN += 1;
                break;
        }
    });

    return counts;
}

function TdCompatibilityCount({
    count,
    dataLabel,
    style,
}: {
    count: number;
    dataLabel: string;
    style: SensorCompatibilityInfo;
}): ReactElement {
    const { Icon, fgColor } = style;
    return (
        <Td dataLabel={dataLabel}>
            {count !== 0 ? (
                <IconText icon={<Icon className={fgColor} />} text={String(count)} />
            ) : (
                count
            )}
        </Td>
    );
}

// Danger, if any sensors are incompatible; otherwise success.
function getSensorCompatibilityVariant(counts: SensorCompatibilityCounts): HealthVariant {
    return counts.INCOMPATIBLE !== 0 ? 'danger' : 'success';
}

function getSensorCompatibilityPhrase(counts: SensorCompatibilityCounts): string {
    return counts.INCOMPATIBLE !== 0
        ? pluralize(counts.INCOMPATIBLE, 'incompatible sensor')
        : 'No incompatible sensors';
}

function SensorCompatibilityHeader({
    counts,
    isFetchingInitialRequest,
}: {
    counts: SensorCompatibilityCounts | null;
    isFetchingInitialRequest: boolean;
}): ReactElement {
    const icon = isFetchingInitialRequest
        ? SpinnerIcon
        : !counts
          ? ErrorIcon
          : healthIconMap[getSensorCompatibilityVariant(counts)];

    const phrase = counts === null ? '' : getSensorCompatibilityPhrase(counts);

    return (
        <Flex grow={{ default: 'grow' }}>
            <FlexItem>{icon}</FlexItem>
            <FlexItem>
                <CardTitle component="h2">Sensor compatibility status</CardTitle>
            </FlexItem>
            {phrase && <FlexItem>{phrase}</FlexItem>}
            <FlexItem align={{ default: 'alignRight' }}>
                <Link to={clustersBasePath}>View clusters</Link>
            </FlexItem>
        </Flex>
    );
}

export type SensorCompatibilityCardProps = {
    clusters: Cluster[];
    isFetchingInitialRequest: boolean;
    errorMessageFetching: string;
};

function SensorCompatibilityCard({
    clusters,
    isFetchingInitialRequest,
    errorMessageFetching,
}: SensorCompatibilityCardProps): ReactElement {
    const counts =
        !isFetchingInitialRequest && !errorMessageFetching
            ? getSensorCompatibilityCounts(clusters)
            : null;

    return (
        <Card isCompact>
            <CardHeader>
                <SensorCompatibilityHeader
                    counts={counts}
                    isFetchingInitialRequest={isFetchingInitialRequest}
                />
            </CardHeader>
            {errorMessageFetching ? (
                <CardBody>
                    <Alert isInline variant="warning" title={errorMessageFetching} component="p" />
                </CardBody>
            ) : (
                counts !== null && (
                    <CardBody>
                        <Table variant="compact" className="pf-v6-u-text-align-right">
                            <Thead>
                                <Tr>
                                    <Th screenReaderText="Clusters" />
                                    <Th>{dataLabelMatched}</Th>
                                    <Th>{dataLabelIncompatible}</Th>
                                    <Th>{dataLabelCompatible}</Th>
                                    <Th>{dataLabelUnknown}</Th>
                                    <Th>Total</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                <Tr>
                                    <Th scope="row">Clusters</Th>
                                    <TdCompatibilityCount
                                        count={counts.MATCHED}
                                        dataLabel={dataLabelMatched}
                                        style={compatibilityStyles.MATCHED}
                                    />
                                    <TdCompatibilityCount
                                        count={counts.INCOMPATIBLE}
                                        dataLabel={dataLabelIncompatible}
                                        style={compatibilityStyles.INCOMPATIBLE}
                                    />
                                    <TdCompatibilityCount
                                        count={counts.COMPATIBLE}
                                        dataLabel={dataLabelCompatible}
                                        style={compatibilityStyles.COMPATIBLE}
                                    />
                                    <TdCompatibilityCount
                                        count={counts.UNKNOWN}
                                        dataLabel={dataLabelUnknown}
                                        style={compatibilityStyles.UNKNOWN}
                                    />
                                    <TdTotal count={clusters.length} />
                                </Tr>
                            </Tbody>
                        </Table>
                    </CardBody>
                )
            )}
        </Card>
    );
}

export default SensorCompatibilityCard;
