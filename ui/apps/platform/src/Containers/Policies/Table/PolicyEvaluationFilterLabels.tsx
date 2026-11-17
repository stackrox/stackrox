import type { ReactElement } from 'react';
import { Label, LabelGroup } from '@patternfly/react-core';
import type { LabelProps } from '@patternfly/react-core';

import type { EvaluationFilter } from 'types/policy.proto';

type PolicyEvaluationFilterLabelsProps = {
    evaluationFilter: EvaluationFilter | null;
};

function PolicyEvaluationFilterLabels({
    evaluationFilter,
}: PolicyEvaluationFilterLabelsProps): ReactElement | null {
    if (!evaluationFilter) {
        return null;
    }

    const labels: {
        text: string;
        color: LabelProps['color'];
    }[] = [];

    if (evaluationFilter.skipContainerTypes?.includes('INIT')) {
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
