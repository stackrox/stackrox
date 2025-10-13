import React, { ReactElement } from 'react';
import { Icon, Spinner } from '@patternfly/react-core';
import {
    CheckCircleIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
    MinusIcon,
} from '@patternfly/react-icons';

// Icon to render while fetching initial request.
export const SpinnerIcon = (
    <Icon size="sm">
        <Spinner />
    </Icon>
);

// Icon to render if request fails.
export const ErrorIcon = <MinusIcon />;

// Icons to render for health after request succeeds.
export const DangerIcon = (
    <Icon>
        <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />
    </Icon>
);
export const SuccessIcon = (
    <Icon>
        <CheckCircleIcon color="var(--pf-v5-global--success-color--100)" />
    </Icon>
);
export const WarningIcon = (
    <Icon>
        <ExclamationTriangleIcon color="var(--pf-v5-global--warning-color--100)" />
    </Icon>
);

export type HealthVariant = 'danger' | 'warning' | 'success';

export const healthIconMap: Record<HealthVariant, ReactElement> = {
    danger: DangerIcon,
    success: SuccessIcon,
    warning: WarningIcon,
};
