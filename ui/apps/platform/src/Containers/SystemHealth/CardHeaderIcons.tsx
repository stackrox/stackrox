import type { ReactElement } from 'react';
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
        <ExclamationCircleIcon color="var(--pf-t--global--icon--color--status--danger--default)" />
    </Icon>
);
export const SuccessIcon = (
    <Icon>
        <CheckCircleIcon color="var(--pf-t--global--icon--color--status--success--default)" />
    </Icon>
);
export const WarningIcon = (
    <Icon>
        <ExclamationTriangleIcon color="var(--pf-t--global--icon--color--status--warning--default)" />
    </Icon>
);

export type HealthVariant = 'danger' | 'warning' | 'success';

export const healthIconMap: Record<HealthVariant, ReactElement> = {
    danger: DangerIcon,
    success: SuccessIcon,
    warning: WarningIcon,
};
