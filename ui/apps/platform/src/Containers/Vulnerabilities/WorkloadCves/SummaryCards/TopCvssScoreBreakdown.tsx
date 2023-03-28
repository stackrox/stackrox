import React from 'react';
import { Card, CardTitle, CardBody, Flex, Tooltip, Text } from '@patternfly/react-core';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

export type TopCvssScoreBreakdownProps = {
    className?: string;
    cvssScore: number;
    vector: string;
};

function TopCvssScoreBreakdown({ className, cvssScore, vector }: TopCvssScoreBreakdownProps) {
    return (
        <Card className={className} isCompact>
            <CardTitle>
                Top CVSS score breakdown{' '}
                <Tooltip content="TODO - Add description for this card">
                    <OutlinedQuestionCircleIcon className="pf-u-display-inline" />
                </Tooltip>
            </CardTitle>
            <CardBody>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                    <Text>{cvssScore.toFixed(1)}</Text>
                    <Text className="pf-u-color-200">{vector}</Text>
                </Flex>
            </CardBody>
        </Card>
    );
}

export default TopCvssScoreBreakdown;
