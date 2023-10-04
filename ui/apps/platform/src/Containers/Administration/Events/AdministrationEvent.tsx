import React, { ReactElement } from 'react';
import {
    BellIcon,
    CheckCircleIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
    InfoCircleIcon,
} from '@patternfly/react-icons';

import {
    AdministrationEventLevel,
    AdministrationEventType,
} from 'services/AdministrationEventsService';

const iconMap: Record<AdministrationEventLevel, ReactElement> = {
    ADMINISTRATION_EVENT_LEVEL_ERROR: (
        <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" />
    ),
    ADMINISTRATION_EVENT_LEVEL_INFO: <InfoCircleIcon color="var(--pf-global--info-color--100)" />,
    ADMINISTRATION_EVENT_LEVEL_SUCCESS: (
        <CheckCircleIcon color="var(--pf-global--success-color--100)" />
    ),
    ADMINISTRATION_EVENT_LEVEL_UNKNOWN: <BellIcon color="var(--pf-global--default-color--200)" />,
    ADMINISTRATION_EVENT_LEVEL_WARNING: (
        <ExclamationTriangleIcon color="var(--pf-global--warning-color--100)" />
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

const typeMap: Record<AdministrationEventType, string> = {
    ADMINISTRATION_EVENT_TYPE_UNKNOWN: 'Unknown',
    ADMINISTRATION_EVENT_TYPE_GENERIC: 'Generic',
    ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE: 'Log',
};

export function getTypeText(type: AdministrationEventType) {
    return typeMap[type] ?? typeMap.ADMINISTRATION_EVENT_TYPE_UNKNOWN;
}
