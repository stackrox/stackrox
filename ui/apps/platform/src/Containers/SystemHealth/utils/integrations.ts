import {
    styleHealthy,
    styleUnhealthy,
    styleUninitialized,
} from 'Containers/Clusters/cluster.helpers';

import { CategoryStyle } from './health';

interface IdNameInterface {
    id: string;
    name: string;
}

export interface IntegrationHealthItem extends IdNameInterface {
    type: string; // differs from type of an integration item: too bad, so sad :(
    status: string;
    errorMessage: string;
    lastTimestamp: string;
}

export interface IntegrationMergedItem extends IntegrationHealthItem {
    label: string;
}

export interface Integration extends IdNameInterface {
    type: string;
}

interface IntegrationsListItem {
    type: string;
    label: string;
}

export type IntegrationStatus = 'HEALTHY' | 'UNINITIALIZED' | 'UNHEALTHY';

export const integrationLabelMap: Record<IntegrationStatus, string> = {
    HEALTHY: 'Healthy',
    UNINITIALIZED: 'Uninitialized',
    UNHEALTHY: 'Unhealthy',
};

export const integrationStyleMap: Record<IntegrationStatus, CategoryStyle> = {
    HEALTHY: styleHealthy,
    UNINITIALIZED: styleUninitialized,
    UNHEALTHY: styleUnhealthy,
};

/*
 * Replace integration health type with integration type and add corresponding label.
 */
export const mergeIntegrationResponses = (
    integrationsHealth: IntegrationHealthItem[],
    integrations: Integration[],
    integrationsList: IntegrationsListItem[]
): IntegrationMergedItem[] => {
    const typeMap: Record<string, string> = {};
    const labelMap: Record<string, string> = {};

    integrations.forEach(({ id, type }) => {
        typeMap[id] = type;
    });
    integrationsList.forEach(({ type, label }) => {
        labelMap[type] = label;
    });

    return integrationsHealth.map((integrationHealthItem) => {
        const type = typeMap[integrationHealthItem.id] ?? '';
        const label = labelMap[type] ?? '';
        return { ...integrationHealthItem, type, label };
    });
};
