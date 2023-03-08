import React from 'react';
import { Card, CardTitle, CardBody, Flex } from '@patternfly/react-core';

import { FixableStatus } from '../types';

export type CvesByStatusSummaryCardProps = {
    cveStatusCounts: Record<FixableStatus, number | 'hidden'>;
    hiddenStatuses: Set<FixableStatus>;
};

function CvesByStatusSummaryCard({
    cveStatusCounts,
    hiddenStatuses,
}: CvesByStatusSummaryCardProps) {
    return (
        <Card>
            <CardTitle>CVEs by status</CardTitle>
            <CardBody>
                <Flex direction={{ default: 'column' }}>
                    <div>
                        {hiddenStatuses.has('Fixable')
                            ? 'Results hidden'
                            : `${cveStatusCounts.Fixable} vulnerabilities with available fixes`}
                    </div>
                    <div>
                        {hiddenStatuses.has('Not fixable')
                            ? 'Results hidden'
                            : `${cveStatusCounts['Not fixable']} vulnerabilities without fixes`}
                    </div>
                </Flex>
            </CardBody>
        </Card>
    );
}

export default CvesByStatusSummaryCard;
