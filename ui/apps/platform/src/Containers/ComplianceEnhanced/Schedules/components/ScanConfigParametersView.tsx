import React from 'react';
import type { ReactElement, ReactNode } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Title,
} from '@patternfly/react-core';

import type { Schedule } from 'services/ComplianceScanConfigurationService';
import { formatScanSchedule } from '../compliance.scanConfigs.utils';

type ScanConfigParametersViewProps = {
    headingLevel: 'h2' | 'h3';
    scanName: string;
    description?: string;
    scanSchedule: Schedule;
    children?: ReactNode;
};

function ScanConfigParametersView({
    description,
    headingLevel,
    scanName,
    scanSchedule,
    children,
}: ScanConfigParametersViewProps): ReactElement {
    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>Parameters</Title>
            <DescriptionList isCompact isHorizontal>
                <DescriptionListGroup>
                    <DescriptionListTerm>Name</DescriptionListTerm>
                    <DescriptionListDescription>{scanName}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Description</DescriptionListTerm>
                    <DescriptionListDescription>
                        {description || <em>No description</em>}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Schedule</DescriptionListTerm>
                    <DescriptionListDescription>
                        {formatScanSchedule(scanSchedule)}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                {children}
            </DescriptionList>
        </Flex>
    );
}

export default ScanConfigParametersView;
