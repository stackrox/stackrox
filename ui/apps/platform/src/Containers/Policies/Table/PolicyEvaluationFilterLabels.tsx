import type { ReactElement } from 'react';
import { Label, LabelGroup } from '@patternfly/react-core';
import type { LabelProps } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { EvaluationFilter } from 'types/policy.proto';

type PolicyEvaluationFilterLabelsProps = {
    evaluationFilter: EvaluationFilter | null;
};

function PolicyEvaluationFilterLabels({
    evaluationFilter,
}: PolicyEvaluationFilterLabelsProps): ReactElement | null {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    if (!evaluationFilter) {
        return null;
    }

    const labels: {
        text: string;
        color: LabelProps['color'];
    }[] = [];

    if (
        isFeatureFlagEnabled('ROX_EVALUATION_FILTER') &&
        isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT') &&
        evaluationFilter.skipContainerTypes?.includes('INIT')
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
