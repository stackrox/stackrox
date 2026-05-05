import { Card, CardBody, DescriptionList, Stack, Title } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { EvaluationFilter, LifecycleStage } from 'types/policy.proto';
import DescriptionListItem from 'Components/DescriptionListItem';

type PolicyFiltersSectionProps = {
    evaluationFilter: EvaluationFilter | undefined;
    lifecycleStages: LifecycleStage[];
};

function getContainerTypeLabel(
    evaluationFilter: EvaluationFilter | undefined,
    lifecycleStages: LifecycleStage[]
): string | null {
    const hasDeployOrRuntime =
        lifecycleStages.includes('DEPLOY') || lifecycleStages.includes('RUNTIME');

    if (!hasDeployOrRuntime) {
        return null;
    }

    const skipped = evaluationFilter?.skipContainerTypes ?? [];
    if (skipped.includes('SKIP_INIT')) {
        return 'Skipping init containers';
    }
    return null;
}

function getImageLayerLabel(evaluationFilter: EvaluationFilter | undefined): string | null {
    switch (evaluationFilter?.skipImageLayers) {
        case 'SKIP_BASE':
            return 'Skip base image layers';
        case 'SKIP_APP':
            return 'Skip application layers';
        default:
            return null;
    }
}

function PolicyFiltersSection({ evaluationFilter, lifecycleStages }: PolicyFiltersSectionProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const containerTypeLabel = isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT')
        ? getContainerTypeLabel(evaluationFilter, lifecycleStages)
        : null;

    const imageLayerLabel = isFeatureFlagEnabled('ROX_POLICY_FILTERS_UI')
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
