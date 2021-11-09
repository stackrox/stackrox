// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/

export type LabelSelectorOperator = 'UNKNOWN' | 'IN' | 'NOT_IN' | 'EXISTS' | 'NOT_EXISTS';

export type LabelSelectorRequirement = {
    key: string;
    op: LabelSelectorOperator;
    values: string[];
};

export type LabelSelector = {
    requirements: LabelSelectorRequirement[];
};

export type MatchLabelsSelector = {
    matchLabels: Record<string, string>;
};
