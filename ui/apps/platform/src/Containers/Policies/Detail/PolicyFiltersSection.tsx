import { Card, CardBody, DescriptionList, Stack, Title } from '@patternfly/react-core';

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

function PolicyFiltersSection({
    evaluationFilter,
    lifecycleStages,
    eventSource,
}: PolicyFiltersSectionProps) {
    const containerTypeLabel = getContainerTypeLabel(
        evaluationFilter,
        lifecycleStages,
        eventSource
    );

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
