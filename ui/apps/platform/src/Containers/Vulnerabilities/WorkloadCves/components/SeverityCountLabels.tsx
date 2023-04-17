import React from 'react';
import { Flex, Label } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';

type SeverityCountLabelsProps = {
    critical: number;
    important: number;
    moderate: number;
    low: number;
};

function SeverityCountLabels({ critical, important, moderate, low }: SeverityCountLabelsProps) {
    const CriticalIcon = SeverityIcons.CRITICAL_VULNERABILITY_SEVERITY;
    const ImportantIcon = SeverityIcons.IMPORTANT_VULNERABILITY_SEVERITY;
    const ModerateIcon = SeverityIcons.MODERATE_VULNERABILITY_SEVERITY;
    const LowIcon = SeverityIcons.LOW_VULNERABILITY_SEVERITY;

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            <Label variant="outline" icon={<CriticalIcon />}>
                {critical}
            </Label>
            <Label variant="outline" icon={<ImportantIcon />}>
                {important}
            </Label>
            <Label variant="outline" icon={<ModerateIcon />}>
                {moderate}
            </Label>
            <Label variant="outline" icon={<LowIcon />}>
                {low}
            </Label>
        </Flex>
    );
}

export default SeverityCountLabels;
