import isEqual from 'lodash/isEqual';
import set from 'lodash/set';
import pluralize from 'pluralize';

import { IntegrationBase } from 'services/IntegrationsService';
import { IntegrationSource, IntegrationType } from 'types/integration';
import { ImageIntegrationCategory } from 'types/imageIntegration.proto';

import { Traits } from 'types/traits.proto';

export type { IntegrationSource, IntegrationType };

export type Integration = {
    type: IntegrationType;
    id: string;
    name: string;
    traits?: Traits;
};

export function getIsAPIToken(source: IntegrationSource, type: IntegrationType): boolean {
    return source === 'authProviders' && type === 'apitoken';
}

export function getIsMachineAccessConfig(
    source: IntegrationSource,
    type: IntegrationType
): boolean {
    return source === 'authProviders' && type === 'machineAccess';
}

export function getIsSignatureIntegration(source: IntegrationSource): boolean {
    return source === 'signatureIntegrations';
}

export function getIsScannerV4(source: IntegrationSource, type: IntegrationType): boolean {
    return source === 'imageIntegrations' && type === 'scannerv4';
}

export function getIsCloudSource(source: IntegrationSource): boolean {
    return source === 'cloudSources';
}

export function getGoogleCredentialsPlaceholder(
    useWorkloadId: boolean,
    updatePassword: boolean
): string {
    if (useWorkloadId) {
        return '';
    }
    if (updatePassword) {
        return 'example,\n{\n  "type": "service_account",\n  "project_id": "123456"\n  ...\n}';
    }
    return 'Currently-stored credentials will be used.';
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

export function getEditDisabledMessage(type) {
    if (type === 'apitoken') {
        return 'This API Token can not be edited. Create a new API Token or delete an existing one.';
    }
    return '';
}

export function transformDurationLongForm(duration: string): string {
    const hours = extractUnitsOfTime(duration, 'h');
    const minutes = extractUnitsOfTime(duration, 'm');
    const seconds = extractUnitsOfTime(duration, 's');
    let result = '';
    if (hours && hours > 0) {
        result += pluralizeUnit(hours, 'hour');
    }
    if (minutes && minutes > 0) {
        result += hours && hours > 0 ? ' ' : '';
        result += pluralizeUnit(minutes, 'minute');
    }
    if (seconds && seconds > 0) {
        result += (hours && hours > 0) || (minutes && minutes > 0) ? ' ' : '';
        result += pluralizeUnit(seconds, 'second');
    }
    return result;
}

function pluralizeUnit(count: number, unit: string): string {
    return `${count} ${pluralize(unit, count)}`;
}

function extractUnitsOfTime(duration: string, unit: string): number {
    const unitRegex = new RegExp(`[0-9]+${unit}`);
    const matchUnits = duration.match(unitRegex);
    if (matchUnits && matchUnits.length !== 0) {
        return parseInt(matchUnits[0].replace(unit, ''));
    }
    return 0;
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

// ["00:00", ..., "23:00"]
export const timesOfDay = new Array(24)
    .fill(1)
    .map((_, t) => `${t.toString().padStart(2, '0')}:00`);

export function backupScheduleDescriptor() {
    return {
        accessor: ({ schedule }) => {
            if (schedule.intervalType === 'WEEKLY') {
                return `Weekly on ${daysOfWeek[schedule.weekly.day]} at ${
                    timesOfDay[schedule.hour]
                } UTC`;
            }
            return `Daily at ${timesOfDay[schedule.hour]} UTC`;
        },
        Header: 'Schedule',
    };
}

// Utilities for image integrations which can have either or both of two categories.

// Categories alternatives correspond to mutually exclusive toggle group items.
type CategoriesAlternatives<
    Category0 extends ImageIntegrationCategory,
    Category1 extends Exclude<ImageIntegrationCategory, Category0>,
> = [
    [[category0: Category0]],
    [[category1: Category1]],
    // The alternative for both categories includes both orders.
    [[category0: Category0, category1: Category1], [category1: Category1, category0: Category0]],
];

// Compiler verifies that first argument of matchCategoriesAlternative method is a category alternative.
type CategoriesAlternative<
    Category0 extends ImageIntegrationCategory,
    Category1 extends Exclude<ImageIntegrationCategory, Category0>,
> = CategoriesAlternatives<Category0, Category1>[number];

function getCategoriesUtils<
    Category0 extends ImageIntegrationCategory,
    Category1 extends Exclude<ImageIntegrationCategory, Category0>,
>([category0, category1]: [Category0, Category1], [text0, text1]: [string, string]) {
    const categoriesAlternatives: CategoriesAlternatives<Category0, Category1> = [
        [[category0]],
        [[category1]],
        [
            [category0, category1],
            [category1, category0],
        ],
    ];

    // For robust behavior in case of unexpected response, provide ternary fallback even though categories limited to Category0 and Category1.
    /* eslint-disable no-nested-ternary */
    return {
        categoriesAlternatives,

        getCategoriesText: (categories: (Category0 | Category1)[]) =>
            categories
                .map((category) =>
                    category === category0 ? text0 : category === category1 ? text1 : category
                )
                .join(' + '),

        matchCategoriesAlternative: (
            categoriesAlternative: CategoriesAlternative<Category0, Category1>,
            categories: (Category0 | Category1)[]
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
