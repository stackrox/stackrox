import type { ReactElement } from 'react';
import { Stack, Title } from '@patternfly/react-core';

import type { LifecycleStage } from 'types/policy.proto';
import type { ProcessViolation, Violation } from 'types/alert.proto';

import DeploytimeMessages from './DeploytimeMessages';
import RuntimeMessages from './RuntimeMessages';

type ViolationDetailsProps = {
    processViolation: ProcessViolation | null;
    lifecycleStage: LifecycleStage;
    violations: Violation[];
};

function ViolationDetails({
    processViolation,
    lifecycleStage,
    violations,
}: ViolationDetailsProps): ReactElement {
    const showRuntimeMessages = processViolation?.processes?.length || lifecycleStage === 'RUNTIME';
    const showDeploytimeMessages = lifecycleStage === 'DEPLOY';
    return (
        <Stack hasGutter>
            <Title headingLevel="h2">Violation events</Title>
            {showRuntimeMessages && (
                <RuntimeMessages
                    processViolation={processViolation}
                    fileAccessViolation={fileAccessViolation}
                    violations={violations}
                />
            )}
            {showDeploytimeMessages && <DeploytimeMessages violations={violations} />}
        </Stack>
    );
}

export default ViolationDetails;
