import { CollectionResponse } from 'services/CollectionsService';
import {
    Collection,
    SelectorField,
    SelectorEntityType,
    isSupportedSelectorField,
    ScopedResourceSelector,
    isByNameSelector,
    isByLabelSelector,
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

const LABEL_SEPARATOR = '=';

/**
 * This function takes a raw `CollectionResponse` from the server and parses it into a representation
 * of a `Collection` that can be supported by the current UI controls. If any incompatibilities are detected
 * it will return a list of validation errors to the caller.
 */
export function parseCollection(data: CollectionResponse): Collection | AggregateError {
    const collection: Collection = {
        name: data.name,
        description: data.description,
        inUse: data.inUse,
        embeddedCollectionIds: data.embeddedCollections.map(({ id }) => id),
        selectorRules: {
            Deployment: null,
            Namespace: null,
            Cluster: null,
        },
    };

    const errors: string[] = [];

    if (data.resourceSelectors.length > 1) {
        errors.push(
            `Multiple 'ResourceSelectors' were found for this collection. Only a single resource selector is supported in the UI. Further validation errors will only apply to the first resource selector in the response.`
        );
    }

    data.resourceSelectors[0]?.rules.forEach((rule) => {
        const entity = fieldToEntityMap[rule.fieldName];
        const field = rule.fieldName;
        const existingEntityField = collection.selectorRules[entity]?.field;
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

        if (!collection.selectorRules[entity]) {
            collection.selectorRules[entity] = {
                field,
                rules: [],
            };
        }
        const selector = collection.selectorRules[entity];

        if (isByLabelSelector(selector)) {
            const firstValue = rule.values[0]?.value;

            if (firstValue && firstValue.includes(LABEL_SEPARATOR)) {
                const key = firstValue.split(LABEL_SEPARATOR)[0] ?? '';
                selector.rules.push({
                    operator: 'OR',
                    key,
                    values: rule.values.map(({ value }) => value.split('=')[1] ?? ''),
                });
            }
        } else if (isByNameSelector(selector)) {
            selector.rules.push({
                operator: 'OR',
                values: rule.values.map(({ value }) => value),
            });
        }
    });

    if (errors.length > 0) {
        return new AggregateError(errors);
    }

    return collection;
}
