import type { ReactElement } from 'react';
import { Flex } from '@patternfly/react-core';

import type { SensorVersionCompatibility } from 'types/cluster.proto';
import {
    getSensorCompatibilityDisplayValue,
    getSensorCompatibilityStyle,
} from '../cluster.helpers';

type SensorCompatibilityProps = {
    compatibility?: SensorVersionCompatibility;
};

function SensorCompatibility({ compatibility }: SensorCompatibilityProps): ReactElement {
    const displayValue = getSensorCompatibilityDisplayValue(compatibility);
    const { Icon, fgColor } = getSensorCompatibilityStyle(compatibility);

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }} gap={{ default: 'gapSm' }}>
            <Icon className={fgColor} />
            <span>{displayValue}</span>
        </Flex>
    );
}

export default SensorCompatibility;
