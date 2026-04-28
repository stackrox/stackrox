import type { ReactElement } from 'react';
import { Label, LabelGroup } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { EvaluationFilter } from 'types/policy.proto';

export type FilterLabel = {
    text: string;
    color: 'blue' | 'orange';
};

type PolicyEvaluationFilterLabelsProps = {
    evaluationFilter: EvaluationFilter | undefined;
    className?: string;
};

function PolicyEvaluationFilterLabels({
    evaluationFilter,
    className,
}: PolicyEvaluationFilterLabelsProps): ReactElement | null {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    if (!evaluationFilter) {
        return null;
    }

    const labels: FilterLabel[] = [];

    if (
        isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT') &&
        evaluationFilter.skipContainerTypes?.includes('SKIP_INIT')
    ) {
        labels.push({ text: 'Skips init', color: 'blue' });
    }

    if (isFeatureFlagEnabled('ROX_IMAGE_LAYER_FILTER')) {
        switch (evaluationFilter.skipImageLayers) {
            case 'SKIP_BASE':
                labels.push({ text: 'App layers only', color: 'orange' });
                break;
            case 'SKIP_APP':
                labels.push({ text: 'Base layers only', color: 'orange' });
                break;
            default:
                break;
        }
    }

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
