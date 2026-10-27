import type { ReactElement } from 'react';
import { Stack, Title } from '@patternfly/react-core';

import type { EvaluationFilter, LifecycleStage, PolicyEventSource } from 'types/policy.proto';
import type { ProcessViolation, Violation } from 'types/alert.proto';

import PolicyFiltersSection from 'Containers/Policies/Detail/PolicyFiltersSection';
import DeploytimeMessages from './DeploytimeMessages';
import RuntimeMessages from './RuntimeMessages';

type ViolationDetailsProps = {
    processViolation: ProcessViolation | null;
    lifecycleStage: LifecycleStage;
    violations: Violation[];
    evaluationFilter: EvaluationFilter | null;
    lifecycleStages: LifecycleStage[];
    eventSource: PolicyEventSource;
};

function ViolationDetails({
    processViolation,
    lifecycleStage,
    violations,
    evaluationFilter,
    lifecycleStages,
    eventSource,
}: ViolationDetailsProps): ReactElement {
    const showRuntimeMessages = processViolation?.processes?.length || lifecycleStage === 'RUNTIME';
    const showDeploytimeMessages = lifecycleStage === 'DEPLOY';
    return (
        <Stack hasGutter>
            <PolicyFiltersSection
                evaluationFilter={evaluationFilter}
                lifecycleStages={lifecycleStages}
                eventSource={eventSource}
            />
            <Title headingLevel="h2">Violation events</Title>
            {showRuntimeMessages && (
                <RuntimeMessages processViolation={processViolation} violations={violations} />
            )}
            {showDeploytimeMessages && <DeploytimeMessages violations={violations} />}
        </Stack>
    );
}

export default ViolationDetails;
