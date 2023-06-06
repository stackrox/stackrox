import React, { ReactElement } from 'react';
import { Spinner } from '@patternfly/react-core';
import {
    CheckCircleIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
    MinusIcon,
} from '@patternfly/react-icons';

// Icon to render while fetching initial request.
export const SpinnerIcon = <Spinner isSVG size="sm" />;

// Icon to render if request fails.
export const ErrorIcon = <MinusIcon />;

// Icons to render for health after request succeeds.
export const DangerIcon = <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" />;
export const SuccessIcon = <CheckCircleIcon color="var(--pf-global--success-color--100)" />;
export const WarningIcon = <ExclamationTriangleIcon color="var(--pf-global--warning-color--100)" />;

export type HealthVariant = 'danger' | 'warning' | 'success';

export const healthIconMap: Record<HealthVariant, ReactElement> = {
    danger: DangerIcon,
    success: SuccessIcon,
    warning: WarningIcon,
};
