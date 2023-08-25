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
import { ReportSnapshot } from 'services/ReportsService.types';
import { getReportStatusText } from '../utils';

export type JobDetailsProps = {
    reportSnapshot: ReportSnapshot;
};

function JobDetails({ reportSnapshot }: JobDetailsProps): ReactElement {
    const { isDownloadAvailable, reportStatus } = reportSnapshot;
    const { reportRequestType, completedAt } = reportStatus;
    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem>
                <Title headingLevel="h3">Job details</Title>
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
