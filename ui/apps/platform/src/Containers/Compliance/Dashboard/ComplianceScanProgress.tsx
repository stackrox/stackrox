import React from 'react';
import { Card, CardBody, CardProps, CardTitle, Progress } from '@patternfly/react-core';

import { ComplianceRunStatusResponse } from './useComplianceRunStatuses';

export type ComplianceDashboardCurrentProps = {
    runs: ComplianceRunStatusResponse['complianceRunStatuses']['runs'];
} & CardProps;

function ComplianceScanProgress({ runs, ...props }: ComplianceDashboardCurrentProps) {
    const unfinishedRunCount = runs.filter((run) => run.state !== 'FINISHED').length;
    const finishedRunCount = runs.length - unfinishedRunCount;

    const title =
        unfinishedRunCount === 0
            ? 'Compliance scanning complete'
            : 'Compliance scanning in progress';

    return (
        <Card {...props}>
            <CardTitle id="compliance-scan-progress-title">{title}</CardTitle>
            <CardBody>
                <Progress
                    aria-labelledby="compliance-scan-progress-title"
                    size="sm"
                    variant={unfinishedRunCount === 0 ? 'success' : undefined}
                    value={finishedRunCount}
                    min={0}
                    max={runs.length}
                    measureLocation={'outside'}
                    label={`${finishedRunCount} of ${runs.length} runs`}
                    valueText={`${finishedRunCount} of ${runs.length} runs`}
                />
            </CardBody>
        </Card>
    );
}

export default ComplianceScanProgress;
