import type { ReactElement } from 'react';
import {
    BellIcon,
    CheckCircleIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
    InfoCircleIcon,
} from '@patternfly/react-icons';
import { Icon } from '@patternfly/react-core';

import type {
    AdministrationEventLevel,
    AdministrationEventType,
} from 'services/AdministrationEventsService';

const iconMap: Record<AdministrationEventLevel, ReactElement> = {
    ADMINISTRATION_EVENT_LEVEL_ERROR: (
        <Icon>
            <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />
        </Icon>
    ),
    ADMINISTRATION_EVENT_LEVEL_INFO: (
        <Icon>
            <InfoCircleIcon color="var(--pf-v5-global--info-color--100)" />
        </Icon>
    ),
    ADMINISTRATION_EVENT_LEVEL_SUCCESS: (
        <Icon>
            <CheckCircleIcon color="var(--pf-v5-global--success-color--100)" />
        </Icon>
    ),
    ADMINISTRATION_EVENT_LEVEL_UNKNOWN: (
        <Icon>
            <BellIcon color="var(--pf-v5-global--default-color--200)" />
        </Icon>
    ),
    ADMINISTRATION_EVENT_LEVEL_WARNING: (
        <Icon>
            <ExclamationTriangleIcon color="var(--pf-v5-global--warning-color--100)" />
        </Icon>
    ),
};

export function getLevelIcon(level: AdministrationEventLevel): ReactElement {
    return iconMap[level] ?? iconMap.ADMINISTRATION_EVENT_LEVEL_UNKNOWN;
}

const textMap: Record<AdministrationEventLevel, string> = {
    ADMINISTRATION_EVENT_LEVEL_ERROR: 'Error',
    ADMINISTRATION_EVENT_LEVEL_INFO: 'Info',
    ADMINISTRATION_EVENT_LEVEL_SUCCESS: 'Success',
    ADMINISTRATION_EVENT_LEVEL_UNKNOWN: 'Unknown',
    ADMINISTRATION_EVENT_LEVEL_WARNING: 'Warning',
};

export function getLevelText(level: AdministrationEventLevel) {
    return textMap[level] ?? textMap.ADMINISTRATION_EVENT_LEVEL_UNKNOWN;
}

type AlertVariant = 'danger' | 'warning' | 'success' | 'info' | 'custom' | undefined;

const variantMap: Record<AdministrationEventLevel, AlertVariant> = {
    ADMINISTRATION_EVENT_LEVEL_ERROR: 'danger',
    ADMINISTRATION_EVENT_LEVEL_INFO: 'info',
    ADMINISTRATION_EVENT_LEVEL_SUCCESS: 'success',
    ADMINISTRATION_EVENT_LEVEL_UNKNOWN: undefined,
    ADMINISTRATION_EVENT_LEVEL_WARNING: 'warning',
};

export function getLevelVariant(level: AdministrationEventLevel) {
    return variantMap[level] ?? variantMap.ADMINISTRATION_EVENT_LEVEL_UNKNOWN;
}

const typeMap: Record<AdministrationEventType, string> = {
    ADMINISTRATION_EVENT_TYPE_UNKNOWN: 'Unknown',
    ADMINISTRATION_EVENT_TYPE_GENERIC: 'Generic',
    ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE: 'Log',
};

export function getTypeText(type: AdministrationEventType) {
    return typeMap[type] ?? typeMap.ADMINISTRATION_EVENT_TYPE_UNKNOWN;
}
