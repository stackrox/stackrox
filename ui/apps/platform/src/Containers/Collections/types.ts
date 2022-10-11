export const selectorEntityTypes = ['Cluster', 'Namespace', 'Deployment'] as const;
export type SelectorEntityType = typeof selectorEntityTypes[number];

export type SelectorField =
    | `${SelectorEntityType}`
    | `${SelectorEntityType} Label`
    | `${SelectorEntityType} Annotation`;

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
 * This type extracts the `fieldName` property from the individual rules and groups them
 * using a single `field` property due to the UI only supporting a single field per entity
 * type in collection rules.
 */
export type ScopedResourceSelector = {
    field: SelectorField;
    rules: Omit<DisjunctionSelectorRule, 'fieldName'>[];
} | null;

/**
 * `Collection` is the front end representation of a valid collection, which is more
 * restricted than Collection objects that can be created via the API.
 */
export type Collection = {
    id?: string;
    name: string;
    description: string;
    inUse: boolean;
    selectorRules: Record<SelectorEntityType, ScopedResourceSelector>;
    embeddedCollectionIds: string[];
};
