import isEqual from 'lodash/isEqual';
import set from 'lodash/set';

import { IntegrationBase } from 'services/IntegrationsService';
import { IntegrationSource, IntegrationType } from 'types/integration';
import { ImageIntegrationCategory } from 'types/imageIntegration.proto';

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

// Utilities for image integrations which can have either or both of two categories.

// Categories alternatives correspond to mutually exclusive toggle group items.
type CategoriesAlternatives<
    Category0 extends ImageIntegrationCategory,
    Category1 extends ImageIntegrationCategory
> = [
    [[category0: Category0]],
    [[category1: Category1]],
    // The alternative for both categories includes both orders.
    [[category0: Category0, category1: Category1], [category1: Category1, category0: Category0]]
];

// Compiler verifies that first argument of matchCategoriesAlternative method is a category alternative.
type CategoriesAlternative<
    Category0 extends ImageIntegrationCategory,
    Category1 extends ImageIntegrationCategory
> = CategoriesAlternatives<Category0, Category1>[number];

function getCategoriesUtils<
    Category0 extends ImageIntegrationCategory,
    Category1 extends ImageIntegrationCategory
>([category0, category1]: [Category0, Category1], [text0, text1]: [string, string]) {
    const categoriesAlternatives: CategoriesAlternatives<Category0, Category1> = [
        [[category0]],
        [[category1]],
        [
            [category0, category1],
            [category1, category0],
        ],
    ];

    // For robust behavior, do not assume that categories from response are limited to Category0 and Category1.
    /* eslint-disable no-nested-ternary */
    return {
        categoriesAlternatives,

        getCategoriesText: (categories: ImageIntegrationCategory[]) =>
            categories
                .map((category) =>
                    category === category0 ? text0 : category === category1 ? text1 : category
                )
                .join(' + '),

        matchCategoriesAlternative: (
            categoriesAlternative: CategoriesAlternative<Category0, Category1>,
            categories: ImageIntegrationCategory[]
        ) =>
            categoriesAlternative.some((categoriesAlternativeItem) =>
                isEqual(categoriesAlternativeItem, categories)
            ),

        validCategories: [category0, category1],
    };
    /* eslint-enable no-nested-ternary */
}

export const categoriesUtilsForClairifyScanner = getCategoriesUtils(
    ['SCANNER', 'NODE_SCANNER'],
    ['Image Scanner', 'Node Scanner']
);

export const categoriesUtilsForRegistryScanner = getCategoriesUtils(
    ['REGISTRY', 'SCANNER'],
    ['Registry', 'Scanner']
);
