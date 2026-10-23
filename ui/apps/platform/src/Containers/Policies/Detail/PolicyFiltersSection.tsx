import { Card, CardBody, DescriptionList, Stack, Title } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { EvaluationFilter, LifecycleStage, PolicyEventSource } from 'types/policy.proto';
import DescriptionListItem from 'Components/DescriptionListItem';
import { containerTypeFilterApplies } from '../policies.utils';

type PolicyFiltersSectionProps = {
    evaluationFilter: EvaluationFilter | null;
    lifecycleStages: LifecycleStage[];
    eventSource: PolicyEventSource;
};

function getContainerTypeLabel(
    evaluationFilter: EvaluationFilter | null,
    lifecycleStages: LifecycleStage[],
    eventSource: PolicyEventSource
): string | null {
    if (!containerTypeFilterApplies(lifecycleStages, eventSource)) {
        return null;
    }

    const skipped = evaluationFilter?.skipContainerTypes ?? [];
    if (skipped.includes('INIT')) {
        return 'Skip init containers';
    }
    return null;
}

function getImageLayerLabel(evaluationFilter: EvaluationFilter | null): string | null {
    switch (evaluationFilter?.skipImageLayers) {
        case 'SKIP_BASE':
            return 'Application layers only';
        case 'SKIP_APP':
            return 'Base layers only';
        default:
            return null;
    }
}

function PolicyFiltersSection({
    evaluationFilter,
    lifecycleStages,
    eventSource,
}: PolicyFiltersSectionProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const containerTypeLabel =
        isFeatureFlagEnabled('ROX_EVALUATION_FILTER') &&
        isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT')
            ? getContainerTypeLabel(evaluationFilter, lifecycleStages, eventSource)
            : null;

    const imageLayerLabel =
        isFeatureFlagEnabled('ROX_IMAGE_LAYER_FILTER')
            ? getImageLayerLabel(evaluationFilter)
            : null;

    if (!containerTypeLabel && !imageLayerLabel) {
        return null;
    }

    return (
        <Stack hasGutter>
            <Title headingLevel="h2">Policy filters</Title>
            <Card>
                <CardBody>
                    <DescriptionList isCompact isHorizontal>
                        {containerTypeLabel && (
                            <DescriptionListItem term="Container types" desc={containerTypeLabel} />
                        )}
                        {imageLayerLabel && (
                            <DescriptionListItem term="Image layers" desc={imageLayerLabel} />
                        )}
                    </DescriptionList>
                </CardBody>
            </Card>
        </Stack>
    );
}

export default PolicyFiltersSection;
