import React from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';
import { ReportJobStatus } from './types';

export type JobStatusPopoverContentProps = {
    statuses: ReportJobStatus[];
};

const contentMap: Record<ReportJobStatus, { label: string; description: string }> = {
    WAITING: {
        label: 'Waiting',
        description: 'The report job is in the queue.',
    },
    PREPARING: {
        label: 'Preparing',
        description: 'The report job is being processed.',
    },
    DOWNLOAD_GENERATED: {
        label: 'Report ready for download',
        description: 'The report is ready and available for download.',
    },
    PARTIAL_SCAN_ERROR_DOWNLOAD: {
        label: 'Partial report ready for download',
        description: 'A report is partially complete and ready for download.',
    },
    EMAIL_DELIVERED: {
        label: 'Report successfully sent',
        description: 'The report was successfully emailed.',
    },
    PARTIAL_SCAN_ERROR_EMAIL: {
        label: 'Partial report successfully sent',
        description: 'A report is partially complete and was successfully emailed.',
    },
    ERROR: {
        label: 'Report failed to generate',
        description: 'There was an issue with the report job. Hover to view the error message.',
    },
};

function JobStatusPopoverContent({ statuses }: JobStatusPopoverContentProps) {
    const content = statuses
        .filter((status) => !!contentMap[status])
        .map((status) => {
            return (
                <FlexItem>
                    <p>
                        <strong>{contentMap[status].label}:</strong>
                    </p>
                    <p>{contentMap[status].description}</p>
                </FlexItem>
            );
        });
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
                <p>
                    Displays the status of your most recent job, whether it is currently running or
                    has completed. The possible statuses are:
                </p>
            </FlexItem>
            {content}
            <FlexItem>
                <p>If no recent jobs are available, &quot;None&quot; will be displayed.</p>
            </FlexItem>
        </Flex>
    );
}

export default JobStatusPopoverContent;
