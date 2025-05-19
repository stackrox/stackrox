import React from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';

function JobStatusPopoverContent() {
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
                <p>
                    Displays the status of your most recent job, whether it is currently running or
                    has completed. The possible statuses are:
                </p>
            </FlexItem>
            <FlexItem>
                <p>
                    <strong>Waiting:</strong>
                </p>
                <p>The report job is in the queue.</p>
            </FlexItem>
            <FlexItem>
                <p>
                    <strong>Preparing:</strong>
                </p>
                <p>The report job is being processed.</p>
            </FlexItem>
            <FlexItem>
                <p>
                    <strong>Report ready for download:</strong>
                </p>
                <p>The report is ready and available for download.</p>
            </FlexItem>
            <FlexItem>
                <p>
                    <strong>Partial report ready for download:</strong>
                </p>
                <p>The report is partially complete and ready for download.</p>
            </FlexItem>
            <FlexItem>
                <p>
                    <strong>Report successfully sent:</strong>
                </p>
                <p>The report has been successfully emailed.</p>
            </FlexItem>
            <FlexItem>
                <p>
                    <strong>Partial report successfully sent:</strong>
                </p>
                <p>The report is partially complete and has been successfully emailed.</p>
            </FlexItem>
            <FlexItem>
                <p>
                    <strong>Report failed to generate:</strong>
                </p>
                <p>There was an issue with the report job. Hover to view the error message.</p>
            </FlexItem>
            <FlexItem>
                <p>If no recent jobs are available, &quot;None&quot; will be displayed.</p>
            </FlexItem>
        </Flex>
    );
}

export default JobStatusPopoverContent;
