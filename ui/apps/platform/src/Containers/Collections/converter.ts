import { CollectionRequest, Collection, SelectorRule } from 'services/CollectionsService';
import { ensureExhaustive, isNonEmptyArray } from 'utils/type.utils';
import {
    ClientCollection,
    SelectorField,
    SelectorEntityType,
    isSupportedSelectorField,
    isByNameField,
    isByLabelField,
    selectorEntityTypes,
    isSelectorField,
    isLabelMatchValue,
    isNameMatchValue,
} from './types';

const fieldToEntityMap: Record<SelectorField, SelectorEntityType> = {
    Deployment: 'Deployment',
    'Deployment Label': 'Deployment',
    'Deployment Annotation': 'Deployment',
    Namespace: 'Namespace',
    'Namespace Label': 'Namespace',
    'Namespace Annotation': 'Namespace',
    Cluster: 'Cluster',
    'Cluster Label': 'Cluster',
    'Cluster Annotation': 'Cluster',
};

export type CollectionParseError = {
    errors: [string, ...string[]];
};

export function isCollectionParseError(
    errorObject: Record<string, unknown>
): errorObject is CollectionParseError {
    return (
        'errors' in errorObject &&
        Array.isArray(errorObject.errors) &&
        isNonEmptyArray(errorObject.errors) &&
        errorObject.errors.every((e) => typeof e === 'string')
    );
}

/**
 * This function takes a raw `CollectionResponse` from the server and parses it into a representation
 * of a `Collection` that can be supported by the current UI controls. If any incompatibilities are detected
 * it will return a list of validation errors to the caller.
 */
export function parseCollection(
    data: Omit<Collection, 'id'>
): ClientCollection | CollectionParseError {
    const collection: ClientCollection = {
        name: data.name,
        description: data.description,
        embeddedCollectionIds: data.embeddedCollections.map(({ id }) => id),
        resourceSelector: {
            Deployment: { type: 'All' },
            Namespace: { type: 'All' },
            Cluster: { type: 'All' },
        },
    };

    const errors: string[] = [];

    if (data.resourceSelectors.length > 1) {
        errors.push(
            `Multiple 'ResourceSelectors' were found for this collection. Only a single resource selector is supported in the UI. Further validation errors will only apply to the first resource selector in the response.`
        );
    }

    data.resourceSelectors[0]?.rules.forEach((rule) => {
        const field = rule.fieldName;
        if (!isSelectorField(field)) {
            errors.push(`An invalid field name was detected. Found field name [${field}].`);
            return;
        }
        const entity = fieldToEntityMap[field];
        const selectorScope = collection.resourceSelector[entity];
        const existingEntityField = selectorScope.type === 'All' ? undefined : selectorScope.field;
        const hasMultipleFieldsForEntity = existingEntityField && existingEntityField !== field;
        const isUnsupportedField = !isSupportedSelectorField(field);
        const isUnsupportedRuleOperator = rule.operator !== 'OR';

        if (hasMultipleFieldsForEntity) {
            errors.push(
                `Each entity type can only contain rules for a single field. A new rule was found for [${entity} -> ${field}], when rules have already been applied for [${entity} -> ${existingEntityField}].`
            );
        }
        if (isUnsupportedField) {
            errors.push(
                `Collection rules for 'Annotation' field names are not supported at this time. Found field name [${field}].`
            );
        }
        if (isUnsupportedRuleOperator) {
            errors.push(
                `Only the disjunction operation ('OR') is currently supported in the front end collection editor. Received an operator of [${rule.operator}].`
            );
        }

        if (hasMultipleFieldsForEntity || isUnsupportedField || isUnsupportedRuleOperator) {
            return;
        }

        if (selectorScope.type === 'All') {
            if (isByLabelField(field)) {
                collection.resourceSelector[entity] = {
                    type: 'ByLabel',
                    field,
                    rules: [],
                };
            } else if (isByNameField(field)) {
                collection.resourceSelector[entity] = {
                    type: 'ByName',
                    field,
                    rule: { operator: 'OR', values: [] },
                };
            }
        }

        const selector = collection.resourceSelector[entity];

        switch (selector.type) {
            case 'ByLabel': {
                const labelMatchValues = rule.values.filter(isLabelMatchValue);
                if (labelMatchValues.length !== rule.values.length) {
                    errors.push(
                        `An invalid match type was detected for a label value: ${JSON.stringify(
                            rule
                        )}`
                    );
                }
                selector.rules.push({
                    operator: 'OR',
                    values: labelMatchValues,
                });
                break;
            }
            case 'ByName': {
                const nameMatchValues = rule.values.filter(isNameMatchValue);
                if (nameMatchValues.length !== rule.values.length) {
                    errors.push(
                        `An invalid match type was detected for a name value: ${JSON.stringify(
                            rule
                        )}`
                    );
                }
                selector.rule.values = nameMatchValues;
                break;
            }
            case 'All':
                // Do nothing
                break;
            default:
                ensureExhaustive(selector);
        }
    });

    if (isNonEmptyArray(errors)) {
        return { errors };
    }

    return collection;
}

/**
 * This function takes the UI supported representation of a `Collection` and generates
 * a server response in the generic `CollectionRequest` format supported by the back end.
 */
export function generateRequest(collection: ClientCollection): CollectionRequest {
    const rules: SelectorRule[] = [];

    selectorEntityTypes.forEach((entityType) => {
        const selector = collection.resourceSelector[entityType];

        switch (selector.type) {
            case 'ByName':
                rules.push({
                    fieldName: selector.field,
                    operator: selector.rule.operator,
                    values: selector.rule.values,
                });
                break;
            case 'ByLabel':
                selector.rules.forEach(({ operator, values }) => {
                    rules.push({
                        fieldName: selector.field,
                        operator,
                        values,
                    });
                });
                break;
            case 'All':
                // Append nothing
                break;
            default:
                ensureExhaustive(selector);
        }
    });

    return {
        name: collection.name,
        description: collection.description,
        embeddedCollectionIds: collection.embeddedCollectionIds,
        resourceSelectors: [{ rules }],
    };
}
