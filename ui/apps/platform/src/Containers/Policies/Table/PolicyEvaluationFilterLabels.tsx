import type { ReactElement } from 'react';
import { Label, LabelGroup } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { EvaluationFilter } from 'types/policy.proto';

type FilterLabel = {
    text: string;
    color: 'blue' | 'orange';
};

type PolicyEvaluationFilterLabelsProps = {
    evaluationFilter: EvaluationFilter | undefined;
};

function PolicyEvaluationFilterLabels({
    evaluationFilter,
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

    if (isFeatureFlagEnabled('ROX_POLICY_FILTERS_UI')) {
        switch (evaluationFilter.skipImageLayers) {
            case 'SKIP_BASE':
                labels.push({ text: 'Skips base', color: 'orange' });
                break;
            case 'SKIP_APP':
                labels.push({ text: 'Skips application', color: 'orange' });
                break;
            default:
                break;
        }
    }

    if (labels.length === 0) {
        return null;
    }

    return (
        <LabelGroup>
            {labels.map(({ text, color }) => (
                <Label key={text} color={color} isCompact variant="outline">
                    {text}
                </Label>
            ))}
        </LabelGroup>
    );
}

export default PolicyEvaluationFilterLabels;
