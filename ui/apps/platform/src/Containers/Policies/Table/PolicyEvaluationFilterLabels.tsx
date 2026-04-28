import type { ReactElement } from 'react';
import { Label, LabelGroup } from '@patternfly/react-core';

import type { EvaluationFilter } from 'types/policy.proto';

export type FilterLabel = {
    text: string;
    color: 'blue' | 'gold';
};

export function getEvaluationFilterLabels(
    evaluationFilter: EvaluationFilter | undefined
): FilterLabel[] {
    if (!evaluationFilter) {
        return [];
    }

    const labels: FilterLabel[] = [];

    if (evaluationFilter.skipContainerTypes?.includes('SKIP_INIT')) {
        labels.push({ text: 'Skips init', color: 'blue' });
    }

    switch (evaluationFilter.skipImageLayers) {
        case 'SKIP_BASE':
            labels.push({ text: 'App layers only', color: 'gold' });
            break;
        case 'SKIP_APP':
            labels.push({ text: 'Base layers only', color: 'gold' });
            break;
        default:
            break;
    }

    return labels;
}

type PolicyEvaluationFilterLabelsProps = {
    evaluationFilter: EvaluationFilter | undefined;
    className?: string;
};

function PolicyEvaluationFilterLabels({
    evaluationFilter,
    className,
}: PolicyEvaluationFilterLabelsProps): ReactElement | null {
    const labels = getEvaluationFilterLabels(evaluationFilter);

    if (labels.length === 0) {
        return null;
    }

    return (
        <LabelGroup className={className}>
            {labels.map(({ text, color }) => (
                <Label key={text} color={color} variant="outline">
                    {text}
                </Label>
            ))}
        </LabelGroup>
    );
}

export default PolicyEvaluationFilterLabels;
