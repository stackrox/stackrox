import React from 'react';
import { Icon } from '@patternfly/react-core';
import { BarsIcon, CheckCircleIcon, SecurityIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import IconText from 'Components/PatternFly/IconText/IconText';

type StatusCountIconProps = {
    text: string;
    status: 'pass' | 'fail' | 'other';
    count: number;
};

function getStatusIcon(status, count) {
    let color = 'var(--pf-v5-global--disabled-color--100)';
    if (count > 0) {
        switch (status) {
            case 'fail':
                color = 'var(--pf-v5-global--danger-color--100)';
                break;
            case 'pass':
                color = 'var(--pf-v5-global--success-color--100)';
                break;
            case 'other':
                color = 'var(--pf-v5-global--disabled-color--100)';
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
        case 'other':
            return <BarsIcon color={color} />;
        default:
            return null;
    }
}

function StatusCountIcon({ text, status, count }: StatusCountIconProps) {
    const icon = <Icon>{getStatusIcon(status, count)}</Icon>;

    return <IconText icon={icon} text={`${count} ${pluralize(text, count)}`} />;
}

export default StatusCountIcon;
