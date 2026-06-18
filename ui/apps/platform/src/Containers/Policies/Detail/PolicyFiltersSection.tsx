import { Card, CardBody, DescriptionList, Stack, Title } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { EvaluationFilter, LifecycleStage } from 'types/policy.proto';
import DescriptionListItem from 'Components/DescriptionListItem';

type PolicyFiltersSectionProps = {
    evaluationFilter: EvaluationFilter | null;
    lifecycleStages: LifecycleStage[];
};

function getContainerTypeLabel(
    evaluationFilter: EvaluationFilter | null,
    lifecycleStages: LifecycleStage[]
): string | null {
    const hasDeployOrRuntime =
        lifecycleStages.includes('DEPLOY') || lifecycleStages.includes('RUNTIME');

    if (!hasDeployOrRuntime) {
        return null;
    }

    const skipped = evaluationFilter?.skipContainerTypes ?? [];
    if (skipped.includes('SKIP_INIT')) {
        return 'Skip init containers';
    }
    return null;
}

function PolicyFiltersSection({ evaluationFilter, lifecycleStages }: PolicyFiltersSectionProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const containerTypeLabel =
        isFeatureFlagEnabled('ROX_EVALUATION_FILTER') &&
        isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT')
            ? getContainerTypeLabel(evaluationFilter, lifecycleStages)
            : null;

    if (!containerTypeLabel) {
        return null;
    }

    return (
        <Stack hasGutter>
            <Title headingLevel="h2">Policy filters</Title>
            <Card>
                <CardBody>
                    <DescriptionList isCompact isHorizontal>
                        <DescriptionListItem term="Container types" desc={containerTypeLabel} />
                    </DescriptionList>
                </CardBody>
            </Card>
        </Stack>
    );
}

export default PolicyFiltersSection;
