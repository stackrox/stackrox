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
    // this type differs from the type of an integration item: too bad, so sad :(
    status: string;
    errorMessage: string;
    lastTimestamp: string;
}

export interface IntegrationMergedItem extends IntegrationHealthItem {
    type: string;
}

export interface Integration extends IdNameInterface {
    type: string;
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
 * Replace integration health type with integration type.
 */
export const mergeIntegrationResponses = (
    integrationsHealth: IntegrationHealthItem[],
    integrations: Integration[]
): IntegrationMergedItem[] => {
    const integrationTypeMap: Record<string, string> = {};

    integrations.forEach(({ id, type }) => {
        integrationTypeMap[id] = type;
    });

    return integrationsHealth.map((integrationHealthItem) => ({
        ...integrationHealthItem,
        type: integrationTypeMap[integrationHealthItem.id] ?? '',
    }));
};
