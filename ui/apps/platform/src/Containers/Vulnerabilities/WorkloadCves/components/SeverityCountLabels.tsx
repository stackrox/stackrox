import React from 'react';
import { Flex, Label } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';

const disabledIconColor = '#d2d2d2';

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
            <Label
                variant="outline"
                className="pf-u-font-weight-bold"
                icon={<CriticalIcon color={critical ? undefined : disabledIconColor} />}
            >
                <span className={critical ? '' : 'pf-u-color-400'}>{critical}</span>
            </Label>
            <Label
                variant="outline"
                className="pf-u-font-weight-bold"
                icon={<ImportantIcon color={important ? undefined : disabledIconColor} />}
            >
                <span className={important ? '' : 'pf-u-color-400'}>{important}</span>
            </Label>
            <Label
                variant="outline"
                className="pf-u-font-weight-bold"
                icon={<ModerateIcon color={moderate ? undefined : disabledIconColor} />}
            >
                <span className={moderate ? '' : 'pf-u-color-400'}>{moderate}</span>
            </Label>
            <Label
                variant="outline"
                className="pf-u-font-weight-bold"
                icon={<LowIcon color={low ? undefined : disabledIconColor} />}
            >
                <span className={low ? '' : 'pf-u-color-400'}>{low}</span>
            </Label>
        </Flex>
    );
}

export default SeverityCountLabels;
