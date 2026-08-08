import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

import type { SensorVersionCompatibility } from 'types/cluster.proto';

import SensorCompatibility from './SensorCompatibility';

type SensorCompatibilitySummaryProps = {
    compatibility?: SensorVersionCompatibility;
    sensorVersion?: string;
    centralVersion: string;
};

function SensorCompatibilitySummary({
    compatibility,
    sensorVersion,
    centralVersion,
}: SensorCompatibilitySummaryProps) {
    return (
        <DescriptionList>
            <DescriptionListGroup>
                <DescriptionListTerm>Status</DescriptionListTerm>
                <DescriptionListDescription>
                    <SensorCompatibility compatibility={compatibility} />
                </DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Sensor version</DescriptionListTerm>
                <DescriptionListDescription>
                    {sensorVersion || 'Unknown'}
                </DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Central version</DescriptionListTerm>
                <DescriptionListDescription>{centralVersion}</DescriptionListDescription>
            </DescriptionListGroup>
        </DescriptionList>
    );
}

export default SensorCompatibilitySummary;
