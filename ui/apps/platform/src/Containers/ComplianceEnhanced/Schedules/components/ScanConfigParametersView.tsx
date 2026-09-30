import type { ReactElement } from 'react';
import { DescriptionList, Flex, Title } from '@patternfly/react-core';

import DetailsDescriptionListGroups from 'Components/Reports/View/DetailsDescriptionListGroups';
import ScheduleDescriptionListGroup from 'Components/Reports/View/ScheduleDescriptionListGroup';
import type { Schedule } from 'types/schedule.proto';

type ScanConfigParametersViewProps = {
    headingLevel: 'h2' | 'h3';
    scanName: string;
    description?: string;
    scanSchedule: Schedule;
};

function ScanConfigParametersView({
    description,
    headingLevel,
    scanName,
    scanSchedule,
}: ScanConfigParametersViewProps): ReactElement {
    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>Parameters</Title>
            <DescriptionList isCompact isHorizontal>
                <DetailsDescriptionListGroups description={description ?? ''} name={scanName} />
                <ScheduleDescriptionListGroup schedule={scanSchedule} />
            </DescriptionList>
        </Flex>
    );
}

export default ScanConfigParametersView;
