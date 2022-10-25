export const selectorEntityTypes = ['Cluster', 'Namespace', 'Deployment'] as const;
export type SelectorEntityType = typeof selectorEntityTypes[number];

export type ByNameSelectorField = `${SelectorEntityType}`;
export type ByLabelSelectorField = `${SelectorEntityType} Label`;
export type ByAnnotationSelectorField = `${SelectorEntityType} Annotation`;

export type SelectorField = ByNameSelectorField | ByLabelSelectorField | ByAnnotationSelectorField;

const byNameRegExp = new RegExp(`^(${selectorEntityTypes.join('|')})$`);
const byLabelRegExp = new RegExp(`^(${selectorEntityTypes.join('|')}) Label$`);
const byAnnotationRegExp = new RegExp(`^(${selectorEntityTypes.join('|')}) Annotation$`);

export function isByNameField(field: SelectorField): field is ByNameSelectorField {
    return byNameRegExp.test(field);
}

export function isByLabelField(field: SelectorField): field is ByLabelSelectorField {
    return byLabelRegExp.test(field);
}

export function isByAnnotationField(field: SelectorField): field is ByAnnotationSelectorField {
    return byAnnotationRegExp.test(field);
}

/**
 * A valid server side `SelectorRule` can use either 'AND' or 'OR' operations to resolve values, but
 * the current UI implementation only supports 'OR'.
 */
export type NameSelectorRule = {
    operator: 'OR';
    values: string[];
};

export type LabelSelectorRule = {
    operator: 'OR';
    key: string;
    values: string[];
};

/**
 * The front end currently only supports rules defined for names and labels, annotations are excluded.
 */
export type SupportedSelectorField = ByNameSelectorField | ByLabelSelectorField;

export function isSupportedSelectorField(field: SelectorField): field is SupportedSelectorField {
    return isByNameField(field) || isByLabelField(field);
}

export type ByNameResourceSelector = {
    field: ByNameSelectorField;
    rule: NameSelectorRule;
};
export type ByLabelResourceSelector = {
    field: ByLabelSelectorField;
    rules: LabelSelectorRule[];
};
export type ScopedResourceSelector =
    | ByNameResourceSelector
    | ByLabelResourceSelector
    | Record<string, never>;

export function isByNameSelector(
    selector: ScopedResourceSelector
): selector is ByNameResourceSelector {
    return isByNameField(selector.field);
}

export function isByLabelSelector(
    selector: ScopedResourceSelector
): selector is ByLabelResourceSelector {
    return isByLabelField(selector.field);
}

/**
 * `Collection` is the front end representation of a valid collection, which is more
 * restricted than Collection objects that can be created via the API.
 */
export type Collection = {
    id?: string;
    name: string;
    description: string;
    inUse: boolean;
    resourceSelectors: Record<SelectorEntityType, ScopedResourceSelector>;
    embeddedCollectionIds: string[];
};
