import type { ReactElement, ReactNode } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Title,
} from '@patternfly/react-core';

import type { Schedule } from 'types/schedule.proto';
import { formatRecurringSchedule } from 'utils/dateUtils';

type ScanConfigParametersViewProps = {
    headingLevel: 'h2' | 'h3';
    scanName: string;
    description?: string;
    scanSchedule: Schedule;
    nodeRoles?: string[];
    children?: ReactNode;
};

function ScanConfigParametersView({
    description,
    headingLevel,
    scanName,
    scanSchedule,
    nodeRoles,
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
                        {formatRecurringSchedule(scanSchedule)}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                {nodeRoles && nodeRoles.length > 0 && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Node roles</DescriptionListTerm>
                        <DescriptionListDescription>
                            {nodeRoles.join(', ')}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                )}
                {children}
            </DescriptionList>
        </Flex>
    );
}

export default ScanConfigParametersView;
