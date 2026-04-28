import type { ReactElement } from 'react';
import { Card, CardBody, DescriptionList } from '@patternfly/react-core';

import type { EvaluationFilter, LifecycleStage } from 'types/policy.proto';
import DescriptionListItem from 'Components/DescriptionListItem';

type PolicyFiltersSectionProps = {
    evaluationFilter: EvaluationFilter | undefined;
    lifecycleStages: LifecycleStage[];
};

function formatContainerTypeFilter(
    evaluationFilter: EvaluationFilter | undefined,
    lifecycleStages: LifecycleStage[]
): string {
    const hasDeployOrRuntime =
        lifecycleStages.includes('DEPLOY') || lifecycleStages.includes('RUNTIME');

    if (!hasDeployOrRuntime) {
        return 'Not applicable (Build lifecycle only)';
    }

    const skipped = evaluationFilter?.skipContainerTypes ?? [];
    if (skipped.includes('SKIP_INIT')) {
        return 'Skipping init containers';
    }
    return 'All container types';
}

function formatImageLayerFilter(evaluationFilter: EvaluationFilter | undefined): string {
    switch (evaluationFilter?.skipImageLayers) {
        case 'SKIP_BASE':
            return 'Application layers only';
        case 'SKIP_APP':
            return 'Base layers only';
        default:
            return 'All layers';
    }
}

function PolicyFiltersSection({
    evaluationFilter,
    lifecycleStages,
}: PolicyFiltersSectionProps): ReactElement {
    return (
        <Card>
            <CardBody>
                <DescriptionList isCompact isHorizontal>
                    <DescriptionListItem
                        term="Container types"
                        desc={formatContainerTypeFilter(evaluationFilter, lifecycleStages)}
                    />
                    <DescriptionListItem
                        term="Image layers"
                        desc={formatImageLayerFilter(evaluationFilter)}
                    />
                </DescriptionList>
            </CardBody>
        </Card>
    );
}

export default PolicyFiltersSection;
