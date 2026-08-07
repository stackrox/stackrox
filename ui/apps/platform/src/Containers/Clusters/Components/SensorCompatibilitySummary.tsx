import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

import useMetadata from 'hooks/useMetadata';
import type { SensorVersionCompatibility } from 'types/cluster.proto';

import SensorVersionRangeChart from './SensorVersionRangeChart';

type SensorCompatibilitySummaryProps = {
    compatibility?: SensorVersionCompatibility;
    sensorVersion: string;
    centralVersion: string;
};

function SensorCompatibilitySummary({
    compatibility,
    sensorVersion,
    centralVersion,
}: SensorCompatibilitySummaryProps) {
    const { compatibleSensorVersions = [] } = useMetadata();

    const showChart =
        compatibility !== undefined &&
        compatibility !== 'SENSOR_VERSION_COMPATIBILITY_UNKNOWN' &&
        compatibleSensorVersions.length > 0;

    return (
        <DescriptionList>
            <DescriptionListGroup>
                <DescriptionListTerm>Version</DescriptionListTerm>
                <DescriptionListDescription>
                    {sensorVersion || 'Unknown'}
                </DescriptionListDescription>
            </DescriptionListGroup>
            {showChart && (
                <DescriptionListGroup>
                    <DescriptionListTerm>Version range</DescriptionListTerm>
                    <DescriptionListDescription>
                        <SensorVersionRangeChart
                            compatibleVersions={compatibleSensorVersions}
                            sensorVersion={sensorVersion}
                            centralVersion={centralVersion}
                            compatibility={compatibility}
                        />
                    </DescriptionListDescription>
                </DescriptionListGroup>
            )}
        </DescriptionList>
    );
}

export default SensorCompatibilitySummary;
