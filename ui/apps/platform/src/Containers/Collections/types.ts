export const selectorEntityTypes = ['Cluster', 'Namespace', 'Deployment'] as const;
export type SelectorEntityType = typeof selectorEntityTypes[number];

export type ByNameSelectorField = `${SelectorEntityType}`;
export type ByLabelSelectorField = `${SelectorEntityType} Label`;
export type ByAnnotationSelectorField = `${SelectorEntityType} Annotation`;

export type SelectorField = ByNameSelectorField | ByLabelSelectorField | ByAnnotationSelectorField;

const byNameRegExp = new RegExp(selectorEntityTypes.join('|'));
const byLabelRegExp = new RegExp(`(${selectorEntityTypes.join('|')}) Label`);
const byAnnotationRegExp = new RegExp(`(${selectorEntityTypes.join('|')}) Annotation`);

export function isByNameSelectorField(field: SelectorField): field is ByNameSelectorField {
    return byNameRegExp.test(field);
}

export function isByLabelSelectorField(field: SelectorField): field is ByLabelSelectorField {
    return byLabelRegExp.test(field);
}

export function isByAnnotationSelectorField(
    field: SelectorField
): field is ByAnnotationSelectorField {
    return byAnnotationRegExp.test(field);
}

type BaseSelectorRule = {
    fieldName: SelectorField;
    values: { value: string }[];
};

export type DisjunctionSelectorRule = BaseSelectorRule & { operator: 'OR' };
export type ConjunctionSelectorRule = BaseSelectorRule & { operator: 'AND' };
/**
 * A valid `SelectorRule` can use either 'AND' or 'OR' operations to resolve values, but
 * since the current UI implementation only supports 'OR', we need to maintain separate types here.
 */
export type SelectorRule = DisjunctionSelectorRule | ConjunctionSelectorRule;

export type ResourceSelector = {
    rules: SelectorRule[];
};

/**
 * The front end currently only supports rules defined for names and labels, annotations are excluded.
 */
export type SupportedSelectorField = ByNameSelectorField | ByLabelSelectorField;

export function isSupportedSelectorField(field: SelectorField): field is SupportedSelectorField {
    return isByNameSelectorField(field) || isByLabelSelectorField(field);
}

/**
 * This type extracts the `fieldName` property from the individual rules and groups them
 * using a single `field` property due to the UI only supporting a single field per entity
 * type in collection rules.
 */
export type ScopedResourceSelector = {
    field: SupportedSelectorField;
    rules: Omit<DisjunctionSelectorRule, 'fieldName'>[];
};

/**
 * `Collection` is the front end representation of a valid collection, which is more
 * restricted than Collection objects that can be created via the API.
 */
export type Collection = {
    id?: string;
    name: string;
    description: string;
    inUse: boolean;
    selectorRules: Record<SelectorEntityType, ScopedResourceSelector | null>;
    embeddedCollectionIds: string[];
};
