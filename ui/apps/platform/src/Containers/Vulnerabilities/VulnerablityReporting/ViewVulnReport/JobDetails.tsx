import React, { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Title,
} from '@patternfly/react-core';

import { getDateTime } from 'utils/dateUtils';
import { ReportStatus } from 'types/reportJob';
import { getReportStatusText } from '../utils';

export type JobDetailsProps = {
    reportStatus: ReportStatus;
    isDownloadAvailable: boolean;
};

function JobDetails({ reportStatus, isDownloadAvailable }: JobDetailsProps): ReactElement {
    const { reportRequestType, completedAt } = reportStatus;
    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem>
                <Title headingLevel="h2">Job details</Title>
            </FlexItem>
            <FlexItem flex={{ default: 'flexNone' }}>
                <DescriptionList
                    isFillColumns
                    columnModifier={{
                        default: '2Col',
                        md: '2Col',
                        sm: '1Col',
                    }}
                >
                    <DescriptionListGroup>
                        <DescriptionListTerm>Status</DescriptionListTerm>
                        <DescriptionListDescription>
                            {getReportStatusText(reportStatus, isDownloadAvailable)}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Run type</DescriptionListTerm>
                        <DescriptionListDescription>
                            {reportRequestType === 'ON_DEMAND' ? 'On demand' : 'Scheduled'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Completed</DescriptionListTerm>
                        <DescriptionListDescription>
                            {completedAt ? getDateTime(completedAt) : '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </FlexItem>
        </Flex>
    );
}

export default JobDetails;
