import React from 'react';
import { Icon } from '@patternfly/react-core';
import { BarsIcon, CheckCircleIcon, SecurityIcon, WrenchIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import IconText from 'Components/PatternFly/IconText/IconText';

import {
    FAILING_VAR_COLOR,
    MANUAL_VAR_COLOR,
    OTHER_VAR_COLOR,
    PASSING_VAR_COLOR,
} from '../compliance.coverage.constants';

type Status = 'pass' | 'fail' | 'manual' | 'other';

type StatusCountIconProps = {
    text: string;
    status: Status;
    count: number;
};

function getStatusIcon(status: Status, count: number) {
    let color = 'var(--pf-v5-global--disabled-color--100)';
    if (count > 0) {
        switch (status) {
            case 'fail':
                color = FAILING_VAR_COLOR;
                break;
            case 'pass':
                color = PASSING_VAR_COLOR;
                break;
            case 'manual':
                color = MANUAL_VAR_COLOR;
                break;
            case 'other':
                color = OTHER_VAR_COLOR;
                break;
            default:
                break;
        }
    }

    switch (status) {
        case 'fail':
            return <SecurityIcon color={color} />;
        case 'pass':
            return <CheckCircleIcon color={color} />;
        case 'manual':
            return <WrenchIcon color={color} />;
        case 'other':
            return (
                <BarsIcon
                    color={color}
                    style={{
                        transform: 'rotate(90deg)',
                    }}
                />
            );
        default:
            return null;
    }
}

function StatusCountIcon({ text, status, count }: StatusCountIconProps) {
    const icon = <Icon>{getStatusIcon(status, count)}</Icon>;

    return <IconText icon={icon} text={`${count} ${pluralize(text, count)}`} />;
}

export default StatusCountIcon;
