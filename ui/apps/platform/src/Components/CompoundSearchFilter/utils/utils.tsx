import type { SearchFilter } from 'types/search';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { getValueByCaseInsensitiveKey, searchValueAsArray } from 'utils/searchUtils';
import { ensureExhaustive } from 'utils/type.utils';

import { convertFromInternalToExternalConditionText } from '../components/SearchFilterConditionText';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
    OnSearchPayload,
    OnSearchPayloadItem,
    OnSearchPayloadItemAdd,
    SelectSearchFilterAttribute,
    SelectSearchFilterGroupedOptions,
    SelectSearchFilterOption,
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

export function getEntityFromConfig(
    config: CompoundSearchFilterConfig,
    entityNameSelected: string | undefined,
    entityNameDefault: string | undefined // when no entity is selected
): CompoundSearchFilterEntity | undefined {
    const entityName = entityNameSelected ?? entityNameDefault;
    const entityFound = config.find((entity) => {
        return entity.displayName === entityName;
    });

    return entityFound ?? config[0]; // default to first entity
}

export function getAttributeFromEntity(
    entity: CompoundSearchFilterEntity | undefined,
    attributeNameSelected: string | undefined,
    attributeNameDefault: string | undefined // when no attribute is selected
): CompoundSearchFilterAttribute | undefined {
    const attributeName = attributeNameSelected ?? attributeNameDefault;
    const attributeFound = entity?.attributes?.find((attribute) => {
        return attribute.displayName === attributeName;
    });

    return attributeFound ?? entity?.attributes?.[0]; // default to first attribute
}

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
    return entity?.attributes ?? [];
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
        const { action } = payloadItem;
        switch (action) {
            case 'APPEND':
            case 'SELECT_INCLUSIVE': {
                const { category, value } = payloadItem;
                const values = searchValueAsArray(searchFilterUpdated[category]);
                searchFilterUpdated[category] = [...values, value];
                break;
            }
            case 'SELECT_EXCLUSIVE': {
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
                ensureExhaustive(action);
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
        case 'APPEND': {
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
        case 'APPEND': // open set of values which analytics might omit
        case 'SELECT_INCLUSIVE': // closed set of values
        case 'SELECT_EXCLUSIVE': // closed set of values
            return true;
        default:
            return false;
    }
}

// Encapsulate inputType and payload for CompoundSearchFilteChips component.

// Information and interaction to render the group of a LabelGroup element.
export type CompoundSearchFilterLabelGroupDescription = {
    label: string; // external text that corresponds to internal value
    payload: OnSearchPayload; // to remove category or value from searchFilter
};

// Information and interaction to render a Label element.
export type CompoundSearchFilterLabelItemDescription = {
    isGlobal?: boolean; // for certain values in AdvancedFilterToolbar.tsx file
} & CompoundSearchFilterLabelGroupDescription;

// Information and interaction to render the label for one or more values of a search filter attribute.
export type CompoundSearchFilterLabelDescription = {
    group: CompoundSearchFilterLabelGroupDescription;
    items: CompoundSearchFilterLabelItemDescription[];
};

export type IsGlobalPredicate = (category: string, value: string) => boolean;

// If attribute has any values in search filter return description else null.
export function getCompoundSearchFilterLabelDescriptionOrNull(
    attribute: CompoundSearchFilterAttribute,
    searchFilter: SearchFilter,
    isGlobalPredicate: IsGlobalPredicate
): CompoundSearchFilterLabelDescription | null {
    const { filterChipLabel, inputType, searchTerm: category } = attribute;

    const payloadItemDeleteCategory: OnSearchPayloadItem = { action: 'DELETE', category };
    const payloadDeleteCategory: OnSearchPayload = [payloadItemDeleteCategory];
    const group: CompoundSearchFilterLabelGroupDescription = {
        label: filterChipLabel,
        payload: payloadDeleteCategory,
    };

    // For example, query might have FIXABLE as key but attribute might have Fixable as key.
    const values = searchValueAsArray(getValueByCaseInsensitiveKey(searchFilter, category));

    switch (inputType) {
        case 'autocomplete':
        case 'condition-number':
        case 'date-picker':
        case 'text': {
            if (values.length === 0) {
                return null;
            }

            return {
                group,
                items: values.map((value) => ({
                    label: value, // external text is same as internal value
                    payload: [{ action: 'REMOVE', category, value }],
                })),
            };
        }
        case 'condition-text': {
            if (values.length === 0) {
                return null;
            }

            const { inputProps } = attribute;
            return {
                group,
                items: values.map((value) => ({
                    label: convertFromInternalToExternalConditionText(inputProps, value),
                    payload: [{ action: 'REMOVE', category, value }],
                })),
            };
        }
        case 'select': {
            if (values.length === 0) {
                return null;
            }

            const options =
                'groupOptions' in attribute.inputProps
                    ? attribute.inputProps.groupOptions.flatMap((group) => group.options)
                    : attribute.inputProps.options;
            return {
                group,
                items: values.map((value) => ({
                    label: getLabelForOption(
                        options.find((option) => option.value === value),
                        value
                    ),
                    payload: [{ action: 'REMOVE', category, value }],
                    isGlobal: isGlobalPredicate(category, value),
                })),
            };
        }
        case 'select-exclusive-single': {
            if (values.length === 0) {
                return null;
            }

            const value = values[0];
            const { inputProps } = attribute;
            const { options } = inputProps;
            return {
                group,
                items: [
                    {
                        label: getLabelForOption(
                            options.find((option) => option.value === value),
                            value
                        ),
                        payload: payloadDeleteCategory,
                    },
                ],
            };
        }
        case 'select-exclusive-double': {
            const { inputProps } = attribute;
            const { category2 } = inputProps;
            const values2 = searchValueAsArray(searchFilter[category2]);

            if (values.length === 0 && values2.length === 0) {
                return null;
            }

            // Assume a value for either category or category2 but not both.
            const value = values.length !== 0 ? values[0] : values2[0];
            const categoryOfValue = values.length !== 0 ? category : category2;
            const { options } = inputProps;
            const payloadDeleteCategories: OnSearchPayload = [
                payloadItemDeleteCategory,
                { action: 'DELETE', category: category2 },
            ];
            return {
                group: {
                    ...group,
                    payload: payloadDeleteCategories,
                },
                items: [
                    {
                        label: getLabelForOption(
                            options.find(
                                (option) =>
                                    option.value === value && option.category === categoryOfValue
                            ),
                            value
                        ),
                        payload: payloadDeleteCategories,
                    },
                ],
            };
        }
        case 'unspecified': {
            if (values.length !== 0) {
                return null;
            }

            // payload is placeholder because only for certain attributes in view-based report
            // For example, Image CVE discovered time: All time
            const { label } = attribute;
            return {
                group,
                items: [{ label, payload: payloadDeleteCategory }],
            };
        }
        default:
            return ensureExhaustive(inputType);
    }
}

// Return internal value if untrusted page address search query does not have a valid option.
function getLabelForOption(option: SelectSearchFilterOption | undefined, value: string) {
    return option ? option.label : value;
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
