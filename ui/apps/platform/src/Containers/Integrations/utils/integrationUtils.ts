import set from 'lodash/set';

import { IntegrationBase } from 'services/IntegrationsService';
import { IntegrationSource, IntegrationType } from 'types/integration';

import integrationsList from './integrationsList';

export type { IntegrationSource, IntegrationType };

export type Integration = {
    type: IntegrationType;
    id: string;
    name: string;
};

export function getIntegrationLabel(source: string, type: string): string {
    const integrationTileLabel = integrationsList[source]?.find(
        (integration) => integration.type === type
    )?.label;
    return typeof integrationTileLabel === 'string' ? integrationTileLabel : '';
}

export function getIsAPIToken(source: IntegrationSource, type: IntegrationType): boolean {
    return source === 'authProviders' && type === 'apitoken';
}

export function getIsClusterInitBundle(source: IntegrationSource, type: IntegrationType): boolean {
    return source === 'authProviders' && type === 'clusterInitBundle';
}

export function getIsSignatureIntegration(source: IntegrationSource): boolean {
    return source === 'signatureIntegrations';
}

/*
 * Return mutated integration with cleared stored credential string properties.
 *
 * Response has '******' for stored credentials, but form values must be empty string unless updating.
 *
 * clearStoredCredentials(integration, ['s3.accessKeyId', 's3.secretAccessKey']);
 * clearStoredCredentials(integration, ['docker.password']);
 * clearStoredCredentials(integration, ['pagerduty.apiKey']);
 */
export function clearStoredCredentials<I extends IntegrationBase>(
    integration: I,
    keyPaths: string[]
): I {
    keyPaths.forEach((keyPath) => {
        set(integration, keyPath, '');
    });
    return integration;
}

export const daysOfWeek = [
    'Sunday',
    'Monday',
    'Tuesday',
    'Wednesday',
    'Thursday',
    'Friday',
    'Saturday',
];

const getTimes = () => {
    const times = ['12:00'];
    for (let i = 1; i <= 11; i += 1) {
        if (i < 10) {
            times.push(`0${i}:00`);
        } else {
            times.push(`${i}:00`);
        }
    }
    return times.map((x) => `${x}AM`).concat(times.map((x) => `${x}PM`));
};

export const timesOfDay = getTimes();
