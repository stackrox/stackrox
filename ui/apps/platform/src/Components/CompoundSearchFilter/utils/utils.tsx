import type { SearchFilter } from 'types/search';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { searchValueAsArray } from 'utils/searchUtils';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
    OnSearchPayload,
    OnSearchPayloadItem,
    OnSearchPayloadItemAdd,
    SelectSearchFilterAttribute,
    SelectSearchFilterGroupedOptions,
    SelectSearchFilterOptions,
} from '../types';

export const conditionMap = {
    'Is greater than': '>',
    'Is greater than or equal to': '>=',
    'Is equal to': '=',
    'Is less than or equal to': '<=',
    'Is less than': '<',
} as const;

export const dateConditionMap = {
    Before: '<',
    On: '', // "=" doesn't work but we can omit the condition to work like an equals
    After: '>',
} as const;

export const conditions = Object.keys(conditionMap) as unknown as (keyof typeof conditionMap)[];

export const dateConditions = Object.keys(
    dateConditionMap
) as unknown as (keyof typeof dateConditionMap)[];

export function getEntity(
    config: CompoundSearchFilterConfig,
    entityName: string
): CompoundSearchFilterEntity | undefined {
    if (!config || !Array.isArray(config)) {
        return undefined;
    }
    const entity = config.find((entity) => {
        return entity.displayName === entityName;
    });
    return entity;
}

export function getAttribute(
    config: CompoundSearchFilterConfig,
    entityName: string,
    attributeName: string
): CompoundSearchFilterAttribute | undefined {
    const entity = getEntity(config, entityName);
    return entity?.attributes?.find((attribute) => {
        return attribute.displayName === attributeName;
    });
}

export function getDefaultEntityName(config: CompoundSearchFilterConfig): string | undefined {
    if (!config || !Array.isArray(config)) {
        return undefined;
    }
    return config?.[0]?.displayName;
}

export function getEntityAttributes(
    config: CompoundSearchFilterConfig,
    entityName: string
): CompoundSearchFilterAttribute[] {
    const entity = getEntity(config, entityName);
    return entity?.attributes || [];
}

export function getDefaultAttributeName(
    config: CompoundSearchFilterConfig,
    entityName: string
): string | undefined {
    const attributes = getEntityAttributes(config, entityName);
    return attributes?.[0]?.displayName;
}

export function ensureConditionNumber(value: unknown): { condition: string; number: number } {
    if (
        typeof value === 'object' &&
        value !== null &&
        'condition' in value &&
        'number' in value &&
        typeof value.condition === 'string' &&
        typeof value.number === 'number'
    ) {
        return {
            condition: value.condition,
            number: value.number,
        };
    }
    return {
        condition: conditions[0],
        number: 0,
    };
}

export function ensureConditionDate(value: unknown): { condition: string; date: string } {
    if (
        typeof value === 'object' &&
        value !== null &&
        'condition' in value &&
        'date' in value &&
        typeof value.condition === 'string' &&
        typeof value.date === 'string'
    ) {
        return {
            condition: value.condition,
            date: value.date,
        };
    }
    return {
        condition: dateConditions[1],
        date: '',
    };
}

export function isSelectType(
    attribute: CompoundSearchFilterAttribute
): attribute is SelectSearchFilterAttribute {
    return attribute.inputType === 'select';
}

export function hasGroupedSelectOptions(
    inputProps: SelectSearchFilterAttribute['inputProps']
): inputProps is SelectSearchFilterGroupedOptions {
    return 'groupOptions' in inputProps;
}

export function hasSelectOptions(
    inputProps: SelectSearchFilterAttribute['inputProps']
): inputProps is SelectSearchFilterOptions {
    return 'options' in inputProps;
}

// Pure function returns searchFilter updated according to payload from interactions.
// Assume that update is needed because payload has already been filtered and is non-empty.
export function updateSearchFilter(
    searchFilter: SearchFilter,
    payload: OnSearchPayload
): SearchFilter {
    const searchFilterUpdated = { ...searchFilter };
    payload.forEach((payloadItem) => {
        switch (payloadItem.action) {
            case 'APPEND_STRING':
            case 'APPEND_TOGGLE': {
                const { category, value } = payloadItem;
                const values = searchValueAsArray(searchFilterUpdated[category]);
                searchFilterUpdated[category] = [...values, value];
                break;
            }
            case 'ASSIGN_SINGLE': {
                const { category, value } = payloadItem;
                searchFilterUpdated[category] = [value];
                break;
            }
            case 'DELETE': {
                const { category } = payloadItem;
                delete searchFilterUpdated[category];
                break;
            }
            case 'REMOVE': {
                const { category, value } = payloadItem;
                const values = searchValueAsArray(searchFilterUpdated[category]);
                searchFilterUpdated[category] = values.filter(
                    (valueInSearchFilter) => valueInSearchFilter !== value
                );
                break;
            }
            default:
                break;
        }
    });

    return searchFilterUpdated;
}

// Pure function returns whether payload item is relevant for updating searchFilter.
export function payloadItemFiltererForUpdating(
    searchFilter: SearchFilter,
    payloadItem: OnSearchPayloadItem
) {
    switch (payloadItem.action) {
        case 'APPEND_STRING': {
            const { category, value } = payloadItem;
            if (value === '') {
                // TODO What is pro and con for search filter input field to prevent empty string?
                return false;
            }

            const values = searchValueAsArray(searchFilter[category]);
            return !values.includes(value); // omit payload item if user entered redundant value
        }
        default:
            return true;
    }
}

// Pure function returns whether payload item is relevant for analytics tracking.
export function payloadItemFiltererForTracking(
    payloadItem: OnSearchPayloadItem
): payloadItem is OnSearchPayloadItemAdd {
    switch (payloadItem.action) {
        case 'APPEND_STRING': // open set of values which analytics might omit
        case 'APPEND_TOGGLE': // closed set of values
        case 'ASSIGN_SINGLE': // closed set of values
            return true;
        default:
            return false;
    }
}

// Given predicate function from useFeatureFlags hook in component
// and searchFilterConfig in which some attributes might have featureFlagDependency property,
// return config to render search filter.
export function getSearchFilterConfigWithFeatureFlagDependency(
    isFeatureFlagEnabled: IsFeatureFlagEnabled,
    searchFilterConfig: CompoundSearchFilterConfig
): CompoundSearchFilterConfig {
    return searchFilterConfig.map((searchFilterEntity) => ({
        ...searchFilterEntity,
        attributes: searchFilterEntity.attributes.filter(({ featureFlagDependency }) => {
            return (
                !Array.isArray(featureFlagDependency) ||
                featureFlagDependency.every(isFeatureFlagEnabled)
            );
        }),
    }));
}
