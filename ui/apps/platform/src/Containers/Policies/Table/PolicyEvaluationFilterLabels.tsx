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

    if (!evaluationFilter || !isFeatureFlagEnabled('ROX_EVALUATION_FILTER')) {
        return null;
    }

    const labels: {
        text: string;
        color: LabelProps['color'];
    }[] = [];

    if (
        isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT') &&
        evaluationFilter.skipContainerTypes?.includes('SKIP_INIT')
    ) {
        labels.push({ text: 'Skips init', color: 'blue' });
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
