import { IntegrationBase } from 'services/IntegrationsService';

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
interface IntegrationsListItem {
    type: string;
    label: string;
}

/*
 * Replace integration health type with integration type and add corresponding label.
 */
export const mergeIntegrationResponses = (
    integrationsHealth: IntegrationHealthItem[],
    integrations: IntegrationBase[],
    descriptors: IntegrationsListItem[]
): IntegrationMergedItem[] => {
    const typeMap: Record<string, string> = {};
    const labelMap: Record<string, string> = {};

    integrations.forEach(({ id, type }) => {
        typeMap[id] = type;
    });
    descriptors.forEach(({ type, label }) => {
        labelMap[type] = label;
    });

    return integrationsHealth.map((integrationHealthItem) => {
        const type = typeMap[integrationHealthItem.id] ?? '';
        const label = labelMap[type] ?? '';
        return { ...integrationHealthItem, type, label };
    });
};
