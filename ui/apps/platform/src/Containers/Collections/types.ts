export const selectorEntityTypes = ['Cluster', 'Namespace', 'Deployment'] as const;
export type SelectorEntityType = (typeof selectorEntityTypes)[number];

export type ByNameSelectorField = `${SelectorEntityType}`;
export type ByLabelSelectorField = `${SelectorEntityType} Label`;
export type ByAnnotationSelectorField = `${SelectorEntityType} Annotation`;

export type SelectorField = ByNameSelectorField | ByLabelSelectorField | ByAnnotationSelectorField;

const byNameRegExp = new RegExp(`^(${selectorEntityTypes.join('|')})$`);
const byLabelRegExp = new RegExp(`^(${selectorEntityTypes.join('|')}) Label$`);
const byAnnotationRegExp = new RegExp(`^(${selectorEntityTypes.join('|')}) Annotation$`);

export function isSelectorField(field: string): field is SelectorField {
    return byNameRegExp.test(field) || byLabelRegExp.test(field) || byAnnotationRegExp.test(field);
}

export function isByNameField(field: SelectorField): field is ByNameSelectorField {
    return byNameRegExp.test(field);
}

export function isByLabelField(field: SelectorField): field is ByLabelSelectorField {
    return byLabelRegExp.test(field);
}

export function isByAnnotationField(field: SelectorField): field is ByAnnotationSelectorField {
    return byAnnotationRegExp.test(field);
}

export const byLabelMatchTypes = ['EXACT'] as const;
export type ByLabelMatchType = (typeof byLabelMatchTypes)[number];
export const byNameMatchType = ['EXACT', 'REGEX'] as const;
export type ByNameMatchType = (typeof byNameMatchType)[number];
export type MatchType = ByNameMatchType | ByLabelMatchType;

/**
 * A valid server side `SelectorRule` can use either 'AND' or 'OR' operations to resolve values, but
 * the current UI implementation only supports 'OR'.
 */
export type NameSelectorRule = {
    operator: 'OR';
    values: { value: string; matchType: ByNameMatchType }[];
};

export type LabelSelectorRule = {
    operator: 'OR';
    values: { value: string; matchType: ByLabelMatchType }[];
};

export function isNameMatchValue(value: {
    value: string;
    matchType: string;
}): value is NameSelectorRule['values'][number] {
    return byNameMatchType.includes(value.matchType as ByNameMatchType);
}

export function isLabelMatchValue(value: {
    value: string;
    matchType: string;
}): value is LabelSelectorRule['values'][number] {
    return byLabelMatchTypes.includes(value.matchType as ByLabelMatchType);
}

/**
 * The front end currently only supports rules defined for names and labels, annotations are excluded.
 */
export type SupportedSelectorField = ByNameSelectorField | ByLabelSelectorField;

export function isSupportedSelectorField(field: SelectorField): field is SupportedSelectorField {
    return isByNameField(field) || isByLabelField(field);
}

export const selectorOptions = ['All', 'ByName', 'ByLabel'] as const;

export type RuleSelectorOption = (typeof selectorOptions)[number];

export type AllResourceSelector = {
    type: 'All';
};
export type ByNameResourceSelector = {
    type: 'ByName';
    field: ByNameSelectorField;
    rule: NameSelectorRule;
};
export type ByLabelResourceSelector = {
    type: 'ByLabel';
    field: ByLabelSelectorField;
    rules: LabelSelectorRule[];
};
export type ScopedResourceSelector =
    | AllResourceSelector
    | ByNameResourceSelector
    | ByLabelResourceSelector;

/**
 * `ClientCollection` is the front end representation of a valid collection, which is more
 * restricted than Collection objects that can be created via the API.
 */
export type ClientCollection = {
    id?: string;
    name: string;
    description: string;
    resourceSelector: Record<SelectorEntityType, ScopedResourceSelector>;
    embeddedCollectionIds: string[];
};
